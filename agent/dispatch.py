#!/usr/bin/env python
"""创建 LiveKit AgentDispatch 规则: 指定 agent 加入指定房间
用法: python agent_dispatch.py <room> [agent_name]
"""
import sys
import time
import jwt
import urllib.request

API_KEY = "devkey"
API_SECRET = "secret"
SERVER = "http://127.0.0.1:7880"

def make_token(room: str):
    now = int(time.time())
    payload = {
        "iss": API_KEY,
        "sub": API_KEY,
        "exp": now + 600,
        "nbf": now - 10,
        "video": {
            # 扁平结构 (VideoGrant 直接序列化): roomAdmin + room 精确匹配目标房间
            "roomAdmin": True,
            "roomCreate": True,
            "roomList": True,
            "room": room,
            "agentAdmin": True,
        },
    }
    return jwt.encode(payload, API_SECRET, algorithm="HS256")

def main():
    room = sys.argv[1] if len(sys.argv) > 1 else "agent-demo"
    agent = sys.argv[2] if len(sys.argv) > 2 else "cam-ai"
    token = make_token(room)
    body = f'{{"agentName": "{agent}", "room": "{room}", "metadata": ""}}'.encode()
    req = urllib.request.Request(
        f"{SERVER}/twirp/livekit.AgentDispatchService/CreateDispatch",
        data=body,
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            print("HTTP", resp.status, resp.read().decode()[:200])
    except Exception as e:
        print("ERR:", e)
        if hasattr(e, "read"):
            print(e.read().decode()[:200])

if __name__ == "__main__":
    main()
