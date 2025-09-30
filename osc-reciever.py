from pythonosc import dispatcher, osc_server
import mido
import argparse

# Default MIDI channel – set this to match Utility > General > MIDI on the SQ
SQ_MIDI_CHANNEL = 1  # (1–16), but mido uses zero-based internally

def get_cc_for_input_fader(input_number: int):
    """Map input channel number (1-48) to the CC number for fader control."""
    if 1 <= input_number <= 48:
        return input_number - 1  # CC = input_number - 1 (0–47)
    else:
        raise ValueError(f"Unsupported input number for fader: {input_number}")

def get_note_for_input_mute(input_number: int):
    """Map input channel number (1-48) to the Note number for mute control."""
    if 1 <= input_number <= 48:
        return input_number - 1  # Note = input_number - 1 (0–47)
    else:
        raise ValueError(f"Unsupported input number for mute: {input_number}")

def osc_handler(address, *args):
    parts = address.strip("/").split("/")
    print("OSC address parts:", parts, "args:", args)

    try:
        if parts[0] == "sq" and parts[1] == "input":
            channel_num = int(parts[2])

            if parts[3] == "fader":
                # Expect args[0] in 0–127 range
                val = int(float(args[0]))
                midi_val = max(0, min(127, val))
                cc_num = get_cc_for_input_fader(channel_num)

                msg = mido.Message(
                    'control_change',
                    channel=SQ_MIDI_CHANNEL - 1,  # zero-based for mido
                    control=cc_num,
                    value=midi_val
                )
                print("Sending CC fader msg:", msg)
                sq5_out.send(msg)

            elif parts[3] == "mute":
                mute_on = bool(int(args[0]))
                note_num = get_note_for_input_mute(channel_num)

                if mute_on:
                    msg = mido.Message(
                        'note_on',
                        channel=SQ_MIDI_CHANNEL - 1,
                        note=note_num,
                        velocity=1  # per CC Translator spec
                    )
                else:
                    msg = mido.Message(
                        'note_off',
                        channel=SQ_MIDI_CHANNEL - 1,
                        note=note_num,
                        velocity=0
                    )
                print("Sending mute msg:", msg)
                sq5_out.send(msg)

    except (IndexError, ValueError) as e:
        print("OSC handler error:", e)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--ip", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=8000)
    parser.add_argument("--sqchan", type=int, default=1,
                        help="SQ MIDI Channel (1-16)")
    args = parser.parse_args()

    global sq5_out, SQ_MIDI_CHANNEL
    SQ_MIDI_CHANNEL = args.sqchan

    print("Available MIDI outputs:")
    counter = 1
    for port in mido.get_output_names():
        print(f"{counter} - {port}")
        counter += 1
    
    number_input = int(input(f"Choose port [{'1' if counter == 2 else f'1-{counter-1}'}] >"))

    # Example: choose the Inputs port explicitly
    sq5_out = mido.open_output(mido.get_output_names()[number_input-1])

    disp = dispatcher.Dispatcher()
    disp.set_default_handler(osc_handler)

    server = osc_server.ThreadingOSCUDPServer((args.ip, args.port), disp)
    print(f"Listening for OSC on {args.ip}:{args.port}")
    print(f"Sending to SQ5 via MIDI port '{sq5_out.name}', MIDI channel {SQ_MIDI_CHANNEL}")
    server.serve_forever()
