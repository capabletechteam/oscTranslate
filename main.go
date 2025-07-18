package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"net"
	"strconv"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/hypebeast/go-osc/osc"
)

var sq5Addr string = "127.0.0.1:51325" // Default to localhost for testing

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("OSC to SQ5 MIDI Translator")
	myWindow.Resize(fyne.NewSize(600, 600))

	logs := widget.NewMultiLineEntry()
	logs.SetMinRowsVisible(10)
	logs.SetText("Application started...\n")

	ipEntry := widget.NewEntry()
	ipEntry.SetText("127.0.0.1")

	oscTab := container.NewVBox(
		widget.NewLabel("SQ5 IP Address:"),
		ipEntry,
		widget.NewButton("Apply", func() {
			sq5Addr = fmt.Sprintf("%s:51325", ipEntry.Text)
			logs.SetText(logs.Text + fmt.Sprintf("SQ5 IP set to %s\n", sq5Addr))
		}),
		widget.NewLabel("OSC Debug Log:"),
		logs,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("OSC to MIDI", oscTab),
		container.NewTabItem("Test Mode", buildTestUI(logs)),
	)

	go startOSCServer(logs)

	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}

func startOSCServer(logs *widget.Entry) {
	addr := "0.0.0.0:8000"
	dispatcher := osc.NewStandardDispatcher()
	dispatcher.AddMsgHandler("*", func(msg *osc.Message) {
		logMsg := fmt.Sprintf("Received OSC: %s %v\n", msg.Address, msg.Arguments)
		logs.SetText(logs.Text + logMsg)
		handleOSCMessage(msg, logs)
	})

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: dispatcher,
	}

	logs.SetText(logs.Text + fmt.Sprintf("Listening for OSC on %s\n", addr))
	if err := server.ListenAndServe(); err != nil {
		logs.SetText(logs.Text + fmt.Sprintf("OSC server error: %v\n", err))
	}
}

func handleOSCMessage(msg *osc.Message, logs *widget.Entry) {
	switch msg.Address {
	case "/fader/1":
		if len(msg.Arguments) > 0 {
			if f, ok := msg.Arguments[0].(float32); ok {
				val := byte(f * 127)
				sendCC(0, 0x00, val)
				logs.SetText(logs.Text + fmt.Sprintf("→ Set Input 1 fader to %d\n", val))
			}
		}
	case "/mute/1":
		if len(msg.Arguments) > 0 {
			if state, ok := msg.Arguments[0].(int32); ok {
				vel := byte(0)
				if state == 1 {
					vel = 127
					logs.SetText(logs.Text + "→ Muted Input 1\n")
				} else {
					logs.SetText(logs.Text + "→ Unmuted Input 1\n")
				}
				sendNoteOn(0, 0x00, vel)
			}
		}
	default:
		logs.SetText(logs.Text + "→ Unmapped OSC path\n")
	}
}

func buildTestUI(logs *widget.Entry) *fyne.Container {
	inputEntry := widget.NewEntry()
	inputEntry.SetPlaceHolder("Input channel (1–48)")

	faderSlider := widget.NewSlider(0, 127)
	faderSlider.Value = 64

	muteCheck := widget.NewCheck("Mute", nil)

	sceneEntry := widget.NewEntry()
	sceneEntry.SetPlaceHolder("Scene number (1–300)")

	return container.NewVBox(
		widget.NewLabel("Test Fader Level"),
		inputEntry,
		faderSlider,
		widget.NewButton("Send Fader", func() {
			ch, err := strconv.Atoi(inputEntry.Text)
			if err != nil || ch < 1 || ch > 48 {
				logs.SetText(logs.Text + "Invalid input channel\n")
				return
			}
			sendCC(0, byte(ch-1), byte(faderSlider.Value))
			logs.SetText(logs.Text + fmt.Sprintf("Sent fader for Input %d to %d\n", ch, int(faderSlider.Value)))
		}),

		widget.NewLabel("Mute/Unmute"),
		muteCheck,
		widget.NewButton("Send Mute", func() {
			ch, err := strconv.Atoi(inputEntry.Text)
			if err != nil || ch < 1 || ch > 48 {
				logs.SetText(logs.Text + "Invalid input channel\n")
				return
			}
			vel := byte(0)
			if muteCheck.Checked {
				vel = 127
			}
			sendNoteOn(0, byte(ch-1), vel)
			logs.SetText(logs.Text + fmt.Sprintf("Mute status sent for Input %d\n", ch))
		}),

		widget.NewLabel("Scene Recall"),
		sceneEntry,
		widget.NewButton("Recall Scene", func() {
			sn, err := strconv.Atoi(sceneEntry.Text)
			if err != nil || sn < 1 || sn > 300 {
				logs.SetText(logs.Text + "Invalid scene number\n")
				return
			}
			sendSceneRecall(0, sn)
			logs.SetText(logs.Text + fmt.Sprintf("Recalled Scene %d\n", sn))
		}),
	)
}

func sendCC(channel, controller, value byte) error {
	msg := []byte{0xB0 | (channel & 0x0F), controller, value}
	return sendMIDIMessage(msg)
}

func sendNoteOn(channel, note, velocity byte) error {
	msg := []byte{0x90 | (channel & 0x0F), note, velocity}
	return sendMIDIMessage(msg)
}

func sendSceneRecall(channel byte, scene int) error {
	bank := byte((scene - 1) / 128)
	program := byte((scene - 1) % 128)
	msg := []byte{
		0xB0 | channel, 0x00, bank,
		0xC0 | channel, program,
	}
	return sendMIDIMessage(msg)
}

func sendMIDIMessage(data []byte) error {
	conn, err := net.Dial("udp", sq5Addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(data)
	fmt.Printf("Sent MIDI: % X to %s\n", data, sq5Addr)
	return err
}
