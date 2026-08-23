"""
CamIngress AI Agent — 监控房间 AI 语音助手 (A) + 字幕 (B) + 视觉理解 (C)

栈: DeepSeek LLM (OpenAI 兼容) + faster-whisper STT (本地) + edge-tts TTS (自写适配器) + silero VAD
运行: cd agent && .venv/Scripts/python.exe agent.py start
环境: agent/.env (见 .env.example)
"""
import os
import io
from dotenv import load_dotenv

load_dotenv(os.path.join(os.path.dirname(__file__), ".env"))

from livekit import agents  # noqa: E402
from livekit.agents import (  # noqa: E402
    APIConnectOptions,
    DEFAULT_API_CONNECT_OPTIONS,
    Agent,
    AgentSession,
    JobContext,
    NOT_GIVEN,
    NotGivenOr,
    RoomInputOptions,
    WorkerOptions,
    cli,
    log,
    stt,
    tts,
    utils,
)
from livekit.agents.stt import StreamAdapter  # noqa: E402
from livekit.plugins import openai, silero  # noqa: E402
from livekit import rtc  # noqa: E402

logger = log.logger


# ---------------------------------------------------------------
# 自写 edge-tts TTS 适配器 (官方无插件, 免费中文音色, MP3→PCM 转码)
# ---------------------------------------------------------------
class _EdgeTTSStream(tts.ChunkedStream):
    """edge-tts 非流式合成: 全部 MP3 块解码后作为单个 AudioFrame 发送"""

    async def _run(self, output_emitter: tts.AudioEmitter) -> None:
        import edge_tts
        import av

        output_emitter.initialize(
            request_id="edge-tts",
            sample_rate=self._tts.sample_rate,
            num_channels=self._tts.num_channels,
            mime_type="audio/pcm",
            stream=False,
        )
        try:
            communicate = edge_tts.Communicate(
                self.input_text, self._tts._voice, rate=self._tts._rate, volume=self._tts._volume
            )
            mp3_chunks: list[bytes] = []
            async for chunk in communicate.stream():
                if chunk["type"] == "audio":
                    mp3_chunks.append(chunk["data"])
            if not mp3_chunks:
                return

            # MP3 → 24kHz 单声道 16bit PCM
            container = av.open(io.BytesIO(b"".join(mp3_chunks)))
            stream = container.streams.audio[0]
            resampler = av.AudioResampler(format="s16", layout="mono", rate=24000)
            pcm = bytearray()
            for frame in container.decode(stream):
                for out in resampler.resample(frame):
                    pcm += out.to_ndarray().tobytes()
            container.close()
            if pcm:
                output_emitter.push(bytes(pcm))
        except Exception:
            logger.exception("edge-tts synthesize failed")
        finally:
            output_emitter.flush()


class EdgeTTS(tts.TTS):
    def __init__(self, voice: str = "zh-CN-XiaoxiaoNeural", rate: str = "+0%", volume: str = "+0%"):
        # edge-tts 输出 24kHz 单声道 MP3, 解码后按 24kHz s16le 封装
        super().__init__(
            capabilities=tts.TTSCapabilities(streaming=False),
            sample_rate=24000,
            num_channels=1,
        )
        self._voice = voice
        self._rate = rate
        self._volume = volume

    def update_options(self, *, voice: str | None = None, rate: str | None = None, volume: str | None = None) -> None:
        if voice:
            self._voice = voice
        if rate:
            self._rate = rate
        if volume:
            self._volume = volume

    def synthesize(
        self, text: str, *, conn_options: APIConnectOptions = DEFAULT_API_CONNECT_OPTIONS
    ) -> tts.ChunkedStream:
        return _EdgeTTSStream(tts=self, input_text=text, conn_options=conn_options)


# ---------------------------------------------------------------
# 自写 faster-whisper STT 适配器 (livekit-agents 1.7 无内置 whisper 插件;
# 用 StreamAdapter 包装非流式识别, 本地 CPU int8 推理)
# ---------------------------------------------------------------
class FasterWhisperSTT(stt.STT):
    def __init__(self, *, model: str = "small", language: str = "zh", device: str = "cpu"):
        from faster_whisper import WhisperModel

        super().__init__(capabilities=stt.STTCapabilities(streaming=False, interim_results=False))
        self._model_name = model
        self._language = language
        # int8 量化: CPU 上速度/内存平衡
        self._model = WhisperModel(model, device=device, compute_type="int8")

    @property
    def model(self) -> str:
        return self._model_name

    @property
    def provider(self) -> str:
        return "faster-whisper"

    async def _recognize_impl(
        self,
        buffer: "utils.AudioBuffer",
        *,
        language: NotGivenOr[str] = NOT_GIVEN,
        conn_options: "APIConnectOptions",
    ) -> stt.SpeechEvent:
        import numpy as np
        from livekit.agents.utils import merge_frames

        # 合并帧 → 16kHz 单声道 float32 (faster-whisper 要求)
        merged = merge_frames(buffer)
        samples = np.frombuffer(merged.data, dtype=np.int16).astype(np.float32) / 32768.0
        if merged.sample_rate != 16000:
            import av
            frame = av.AudioFrame.from_ndarray(
                np.expand_dims(samples, 0) * 32768.0, format="s16", layout="mono"
            )
            frame.sample_rate = merged.sample_rate
            resampler = av.AudioResampler(format="s16", layout="mono", rate=16000)
            out_frames = list(resampler.resample(frame))
            if not out_frames:
                return stt.SpeechEvent(type=stt.SpeechEventType.FINAL_TRANSCRIPT, request_id="")
            samples = np.frombuffer(out_frames[0].to_ndarray().tobytes(), dtype=np.int16).astype(np.float32) / 32768.0

        segments, _ = self._model.transcribe(
            samples, language=self._language, beam_size=1, vad_filter=False
        )
        text = "".join(s.text for s in segments).strip()
        if not text:
            return stt.SpeechEvent(type=stt.SpeechEventType.FINAL_TRANSCRIPT, request_id="")
        return stt.SpeechEvent(
            type=stt.SpeechEventType.FINAL_TRANSCRIPT,
            request_id="",
            alternatives=[
                stt.SpeechData(
                    language=self._language,
                    text=text,
                    start_time=0.0,
                    end_time=merged.duration,
                )
            ],
        )


# ---------------------------------------------------------------
# 主 Agent — 监控房间 AI 值班员
# ---------------------------------------------------------------
class CamAgent(Agent):
    def __init__(self):
        super().__init__(
            instructions=(
                "你是摄像头监控系统的 AI 值班助手。"
                "用户可能就监控画面或安防情况向你提问。"
                "回答要简洁(通常 1-2 句), 用中文, 语气自然友好。"
            ),
            stt=StreamAdapter(stt=FasterWhisperSTT(), vad=silero.VAD.load()),
            llm=openai.LLM(
                base_url=os.environ.get("LLM_BASE_URL", "https://api.deepseek.com"),
                api_key=os.environ["LLM_API_KEY"],
                model=os.environ.get("LLM_MODEL", "deepseek-chat"),
            ),
            tts=EdgeTTS(voice=os.environ.get("TTS_VOICE", "zh-CN-XiaoxiaoNeural")),
            vad=silero.VAD.load(),
        )


async def entrypoint(ctx: JobContext):
    await ctx.connect()
    # livekit-agents 1.7: AgentSession 不直接收 agent, 通过 start(agent=...) 启动
    session = AgentSession()
    await session.start(
        agent=CamAgent(),
        room=ctx.room,
        room_input_options=RoomInputOptions(noise_cancellation=False),
    )


if __name__ == "__main__":
    # port=0: 自动分配 worker HTTP 端口 (prod 默认 8081 与 docker ingress 的 WHIP 端口冲突)
    # load_fnc=0: 默认 CPU 负载判定在个别僵尸进程占核时会永久标记不可用, 改恒可用
    cli.run_app(
        WorkerOptions(
            entrypoint_fnc=entrypoint,
            agent_name="cam-ai",
            port=0,
            load_fnc=lambda: 0.0,
        )
    )
