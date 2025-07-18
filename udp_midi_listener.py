import socket

UDP_IP = "0.0.0.0"
UDP_PORT = 51325

print(f"SQ5 Simulator listening on {UDP_IP}:{UDP_PORT}...\n")

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind((UDP_IP, UDP_PORT))

def describe_sq5_action(data):
    if len(data) < 2:
        return "Incomplete MIDI message"

    status = data[0]
    channel = status & 0x0F
    command = status & 0xF0

    if command == 0xB0 and len(data) >= 3:  # Control Change
        cc = data[1]
        value = data[2]
        percent = int((value / 127) * 100)

        if 0x00 <= cc <= 0x1F:
            input_ch = cc + 1
            return f"Set Input {input_ch} fader to {percent}%"
        elif 0x20 <= cc <= 0x3F:
            ch = cc - 0x20 + 1
            return f"Set Mix Send to Mix 1 from Input {ch} to {percent}%"
        elif 0x40 <= cc <= 0x5F:
            ch = cc - 0x40 + 1
            return f"Set Mix Send to Mix 2 from Input {ch} to {percent}%"
        elif cc == 0x60:
            return f"Set FX Return 1 fader to {percent}%"
        elif cc == 0x61:
            return f"Set FX Return 2 fader to {percent}%"
        elif cc == 0x62:
            return f"Set FX Return 3 fader to {percent}%"
        elif cc == 0x63:
            return f"Set FX Return 4 fader to {percent}%"
        elif cc == 0x7F:
            return f"Set Main LR fader to {percent}%"
        else:
            return f"Unknown CC {cc:02X} with value {value}"

    elif command == 0x90 and len(data) >= 3:  # Note On = Mute
        note = data[1]
        vel = data[2]
        state = "Mute" if vel > 0 else "Unmute"

        if 0x00 <= note <= 0x1F:
            return f"{state} Input {note + 1}"
        elif 0x20 <= note <= 0x2F:
            return f"{state} Mix {note - 0x20 + 1}"
        elif 0x30 <= note <= 0x33:
            return f"{state} FX Return {note - 0x30 + 1}"
        elif note == 0x3F:
            return f"{state} Main LR"
        else:
            return f"{state} Unknown Note {note:02X}"

    elif command == 0xC0 and len(data) >= 2:  # Program Change = Scene Recall
        scene = data[1] + 1
        return f"Recall Scene {scene}"

    else:
        return f"Unknown or unsupported MIDI command: {status:02X}"

while True:
    data, addr = sock.recvfrom(1024)
    hex_string = ' '.join(f'{byte:02X}' for byte in data)
    print(f"From {addr}: {hex_string}")
    print("  ↳", describe_sq5_action(data))
