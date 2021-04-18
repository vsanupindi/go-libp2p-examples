package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ChatUI is a Text User Interface (TUI) for a ChatRoom.
// The Run method will draw the UI to the terminal in "fullscreen"
// mode. You can quit with Ctrl-C, or by typing "/quit" into the
// chat prompt.
type ChatUI struct {
	cr        *ChatRoom
	app       *tview.Application
	peersList *tview.TextView

	msgW    io.Writer
	inputCh chan string
	doneCh  chan struct{}
}

type OpinionMessage struct {
	user    string
	stock   string
	numeric int
	opinion string
}

var localOpinions []OpinionMessage // locally stored list of recieved opinions

// NewChatUI returns a new ChatUI struct that controls the text UI.
// It won't actually do anything until you call Run().
func NewChatUI(cr *ChatRoom) *ChatUI {
	app := tview.NewApplication()

	// make a text view to contain our chat messages
	msgBox := tview.NewTextView()
	msgBox.SetDynamicColors(true)
	msgBox.SetBorder(true)
	msgBox.SetTitle(fmt.Sprintf("Room: %s", cr.roomName))

	// text views are io.Writers, but they don't automatically refresh.
	// this sets a change handler to force the app to redraw when we get
	// new messages to display.
	msgBox.SetChangedFunc(func() {
		app.Draw()
	})

	// an input field for typing messages into
	inputCh := make(chan string, 32)
	input := tview.NewInputField().
		SetLabel(cr.nick + " > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack)

	// the done func is called when the user hits enter, or tabs out of the field
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			// we don't want to do anything if they just tabbed away
			return
		}
		line := input.GetText()
		if len(line) == 0 {
			// ignore blank lines
			return
		}

		// bail if requested
		if line == "/quit" {
			app.Stop()
			return
		}

		if line == "/share" {
			var dirPath = "stocks" + "-" + cr.nick + "/"
			files, err := ioutil.ReadDir(dirPath)
			if err != nil {
				panic(err)
			}
			for _, file := range files {
				if file.Name() == cr.roomName+".txt" {
					content, err := ioutil.ReadFile(dirPath + file.Name())
					if err != nil {
						panic(err)
					}
					var opinion = string(content)
					line = opinion
				}
			}
		}

		// send the line onto the input chan and reset the field text
		inputCh <- line
		input.SetText("")
	})

	// make a text view to hold the list of peers in the room, updated by ui.refreshPeers()
	peersList := tview.NewTextView()
	peersList.SetBorder(true)
	peersList.SetTitle("Peers")
	peersList.SetChangedFunc(func() { app.Draw() })

	// chatPanel is a horizontal box with messages on the left and peers on the right
	// the peers list takes 20 columns, and the messages take the remaining space
	chatPanel := tview.NewFlex().
		AddItem(msgBox, 0, 1, false).
		AddItem(peersList, 20, 1, false)

	// flex is a vertical box with the chatPanel on top and the input field at the bottom.

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatPanel, 0, 1, false).
		AddItem(input, 1, 1, true)

	app.SetRoot(flex, true)

	return &ChatUI{
		cr:        cr,
		app:       app,
		peersList: peersList,
		msgW:      msgBox,
		inputCh:   inputCh,
		doneCh:    make(chan struct{}, 1),
	}
}

//Clay added this function
/*func (ui *ChatUI) postOpinion() {
	var dirPath = "stocks" + "-" + ui.cr.nick + "/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		content, err := ioutil.ReadFile(dirPath + file.Name())
		if err != nil {
			panic(err)
		}
		var opinion = string(content)

		error := ui.cr.Publish(opinion)
		if error != nil {
			panic(error)
		}
		ui.displaySelfMessage(opinion)

		// vaishu
		var writePath = "sentiments/"
		f, err := os.OpenFile(writePath+file.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		b, err := ioutil.ReadFile(writePath + file.Name())
		if err != nil {
			panic(err)
		}
		s := string(b)
		// //check whether s contains substring text
		if !strings.Contains(s, ui.cr.nick) {
			if _, err = f.WriteString(ui.cr.nick + " - " + opinion + "\n"); err != nil {
				panic(err)
			}
		}
		// end vaishu

		//fmt.Println(string(content))

		// fmt.Println(file.Name())
		// stockFileName = file.Name()
		// var extension = filepath.Ext(stockFileName)
		// name = stockFileName[0 : len(stockFileName)-len(extension)]
		// fmt.Println(name)
	}
}*/

// Run starts the chat event loop in the background, then starts
// the event loop for the text UI.
func (ui *ChatUI) Run() error {
	go ui.handleEvents()

	defer ui.end()

	// vaishu commented this out - not anymore
	//ui.postOpinion()

	return ui.app.Run()
}

// end signals the event loop to exit gracefully
func (ui *ChatUI) end() {
	ui.doneCh <- struct{}{}
}

// refreshPeers pulls the list of peers currently in the chat room and
// displays the last 8 chars of their peer id in the Peers panel in the ui.
func (ui *ChatUI) refreshPeers() {
	peers := ui.cr.ListPeers()

	// clear is not threadsafe so we need to take the lock.
	ui.peersList.Lock()
	ui.peersList.Clear()
	ui.peersList.Unlock()

	for _, p := range peers {
		fmt.Fprintln(ui.peersList, shortID(p))
	}

	ui.app.Draw()
}

// vaishu
func getOpinionVal(cm *ChatMessage) bool{
	r := reflect.ValueOf(cm)
	f := reflect.Indirect(r).FieldByName("Opinion")
	return bool(f.Bool())
}

func getMessageVal(cm *ChatMessage) string{
	r := reflect.ValueOf(cm)
	f := reflect.Indirect(r).FieldByName("Message")
	return string(f.String())
}

// displayChatMessage writes a ChatMessage from the room to the message window,
// with the sender's nick highlighted in green.
//THIS IS WHERE WE CHECK FOR OPINIONS AND SAVE THEM IF TRUE
func (ui *ChatUI) displayChatMessage(cm *ChatMessage) {
	prompt := withColor("green", fmt.Sprintf("<%s>:", cm.SenderNick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, cm.Message)
	if cm.Opinion == true {
		// can do something here for storing the data?
	}
	//localOpinions = append(localOpinions, [Opinion Message Struct Variable Goes Here]})
	//fmt.Fprintf(ui.msgW, getMessageVal(cm))
	//fmt.Fprintf(ui.msgW, "%t \n", getOpinionVal(cm))
}

// displaySelfMessage writes a message from ourself to the message window,
// with our nick highlighted in yellow.
func (ui *ChatUI) displaySelfMessage(msg string) {
	prompt := withColor("yellow", fmt.Sprintf("<%s>:", ui.cr.nick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, msg)
}

// handleEvents runs an event loop that sends user input to the chat room
// and displays messages received from the chat room. It also periodically
// refreshes the list of peers in the UI.
func (ui *ChatUI) handleEvents() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()

	for {
		select {
		case input := <-ui.inputCh:
			// when the user types in a line, publish it to the chat room and print to the message window
			// vaishu
			// vaishu - here we need to see if the inputCh is JSON or not and
			// use Publish or PublishOpinion respectively
			if strings.Contains(input, "{"){
				err := ui.cr.PublishOpinion(input)
				if err != nil {
					printErr("publish error: %s", err)
				}
				//ui.displaySelfMessage(input)
			} else {
				err := ui.cr.Publish(input)
				if err != nil {
					printErr("publish error: %s", err)
				}
				//ui.displaySelfMessage("Here")
				ui.displaySelfMessage(input)
		}
			// end vaishu

		case m := <-ui.cr.Messages:
			// when we receive a message from the chat room, print it to the message window
			ui.displayChatMessage(m)
			// vaishu - here we need to go to displayChatMessage

		case <-peerRefreshTicker.C:
			// refresh the list of peers in the chat room periodically
			ui.refreshPeers()

		case <-ui.cr.ctx.Done():
			return

		case <-ui.doneCh:
			return
		}
	}
}

// withColor wraps a string with color tags for display in the messages text box.
func withColor(color, msg string) string {
	return fmt.Sprintf("[%s]%s[-]", color, msg)
}
