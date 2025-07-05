package tui

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode/internal/app"
	"github.com/sst/opencode/internal/commands"
	"github.com/sst/opencode/internal/completions"
	"github.com/sst/opencode/internal/components/chat"
	"github.com/sst/opencode/internal/components/dialog"
	"github.com/sst/opencode/internal/components/modal"
	"github.com/sst/opencode/internal/components/status"
	"github.com/sst/opencode/internal/components/toast"
	"github.com/sst/opencode/internal/layout"
	"github.com/sst/opencode/internal/styles"
	"github.com/sst/opencode/internal/theme"
	"github.com/sst/opencode/internal/util"
)

// InterruptDebounceTimeoutMsg is sent when the interrupt key debounce timeout expires
type InterruptDebounceTimeoutMsg struct{}

// InterruptKeyState tracks the state of interrupt key presses for debouncing
type InterruptKeyState int

const (
	InterruptKeyIdle InterruptKeyState = iota
	InterruptKeyFirstPress
)

const interruptDebounceTimeout = 1 * time.Second

type appModel struct {
	width, height        int
	app                  *app.App
	modal                layout.Modal
	status               status.StatusComponent
	editor               chat.EditorComponent
	messages             chat.MessagesComponent
	completions          dialog.CompletionDialog
	completionManager    *completions.CompletionManager
	showCompletionDialog bool
	leaderBinding        *key.Binding
	isLeaderSequence     bool
	toastManager         *toast.ToastManager
	interruptKeyState    InterruptKeyState
	lastScroll           time.Time
}

func (a appModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	// https://github.com/charmbracelet/bubbletea/issues/1440
	// https://github.com/sst/opencode/issues/127
	if !util.IsWsl() {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	cmds = append(cmds, a.app.InitializeProvider())
	cmds = append(cmds, a.editor.Init())
	cmds = append(cmds, a.messages.Init())
	cmds = append(cmds, a.status.Init())
	cmds = append(cmds, a.completions.Init())
	cmds = append(cmds, a.toastManager.Init())

	// Check if we should show the init dialog
	cmds = append(cmds, func() tea.Msg {
		shouldShow := a.app.Info.Git && a.app.Info.Time.Initialized > 0
		return dialog.ShowInitDialogMsg{Show: shouldShow}
	})

	return tea.Batch(cmds...)
}

var BUGGED_SCROLL_KEYS = map[string]bool{
	"0": true,
	"1": true,
	"2": true,
	"3": true,
	"4": true,
	"5": true,
	"6": true,
	"7": true,
	"8": true,
	"9": true,
	"M": true,
	"m": true,
	"[": true,
	";": true,
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		keyString := msg.String()
		if time.Since(a.lastScroll) < time.Millisecond*100 && BUGGED_SCROLL_KEYS[keyString] {
			return a, nil
		}

		// 1. Handle active modal
		if a.modal != nil {
			switch keyString {
			// Escape always closes current modal
			case "esc", "ctrl+c":
				cmd := a.modal.Close()
				a.modal = nil
				return a, cmd
			}

			// Pass all other key presses to the modal
			updatedModal, cmd := a.modal.Update(msg)
			a.modal = updatedModal.(layout.Modal)
			return a, cmd
		}

		// 2. Check for commands that require leader
		if a.isLeaderSequence {
			matches := a.app.Commands.Matches(msg, a.isLeaderSequence)
			a.isLeaderSequence = false
			if len(matches) > 0 {
				return a, util.CmdHandler(commands.ExecuteCommandsMsg(matches))
			}
		}

		// 3. Handle completions trigger
		if keyString == "/" && !a.showCompletionDialog {
			a.showCompletionDialog = true

			initialValue := "/"
			currentInput := a.editor.Value()

			// if the input doesn't end with a space,
			// then we want to include the last word
			// (ie, `packages/`)
			if !strings.HasSuffix(currentInput, " ") {
				words := strings.Split(a.editor.Value(), " ")
				if len(words) > 0 {
					lastWord := words[len(words)-1]
					lastWord = strings.TrimSpace(lastWord)
					initialValue = lastWord + "/"
				}
			}

			updated, cmd := a.completions.Update(
				app.CompletionDialogTriggeredMsg{
					InitialValue: initialValue,
				},
			)
			a.completions = updated.(dialog.CompletionDialog)
			cmds = append(cmds, cmd)

			updated, cmd = a.editor.Update(msg)
			a.editor = updated.(chat.EditorComponent)
			cmds = append(cmds, cmd)

			updated, cmd = a.updateCompletions(msg)
			a.completions = updated.(dialog.CompletionDialog)
			cmds = append(cmds, cmd)

			return a, tea.Sequence(cmds...)
		}

		if a.showCompletionDialog {
			switch keyString {
			case "tab", "enter", "esc", "ctrl+c":
				updated, cmd := a.updateCompletions(msg)
				a.completions = updated.(dialog.CompletionDialog)
				cmds = append(cmds, cmd)
				return a, tea.Batch(cmds...)
			}

			updated, cmd := a.editor.Update(msg)
			a.editor = updated.(chat.EditorComponent)
			cmds = append(cmds, cmd)

			updated, cmd = a.updateCompletions(msg)
			a.completions = updated.(dialog.CompletionDialog)
			cmds = append(cmds, cmd)

			return a, tea.Batch(cmds...)
		}

		// 4. Maximize editor responsiveness for printable characters
		if msg.Text != "" {
			updated, cmd := a.editor.Update(msg)
			a.editor = updated.(chat.EditorComponent)
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}

		// 5. Check for leader key activation
		if a.leaderBinding != nil &&
			!a.isLeaderSequence &&
			key.Matches(msg, *a.leaderBinding) {
			a.isLeaderSequence = true
			return a, nil
		}

		// 6. Handle interrupt key debounce for session interrupt
		interruptCommand := a.app.Commands[commands.SessionInterruptCommand]
		if interruptCommand.Matches(msg, a.isLeaderSequence) && a.app.IsBusy() {
			switch a.interruptKeyState {
			case InterruptKeyIdle:
				// First interrupt key press - start debounce timer
				a.interruptKeyState = InterruptKeyFirstPress
				a.editor.SetInterruptKeyInDebounce(true)
				return a, tea.Tick(interruptDebounceTimeout, func(t time.Time) tea.Msg {
					return InterruptDebounceTimeoutMsg{}
				})
			case InterruptKeyFirstPress:
				// Second interrupt key press within timeout - actually interrupt
				a.interruptKeyState = InterruptKeyIdle
				a.editor.SetInterruptKeyInDebounce(false)
				return a, util.CmdHandler(commands.ExecuteCommandMsg(interruptCommand))
			}
		}

		// 7. Check again for commands that don't require leader (excluding interrupt when busy)
		matches := a.app.Commands.Matches(msg, a.isLeaderSequence)
		if len(matches) > 0 {
			// Skip interrupt key if we're in debounce mode and app is busy
			if interruptCommand.Matches(msg, a.isLeaderSequence) && a.app.IsBusy() && a.interruptKeyState != InterruptKeyIdle {
				return a, nil
			}
			return a, util.CmdHandler(commands.ExecuteCommandsMsg(matches))
		}

		// 7. Fallback to editor. This is for other characters
		// like backspace, tab, etc.
		updatedEditor, cmd := a.editor.Update(msg)
		a.editor = updatedEditor.(chat.EditorComponent)
		return a, cmd
	case tea.MouseWheelMsg:
		a.lastScroll = time.Now()
		if a.modal != nil {
			return a, nil
		}
		updated, cmd := a.messages.Update(msg)
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	case tea.BackgroundColorMsg:
		styles.Terminal = &styles.TerminalInfo{
			Background:       msg.Color,
			BackgroundIsDark: msg.IsDark(),
		}
		slog.Debug("Background color", "color", msg.String(), "isDark", msg.IsDark())
		return a, func() tea.Msg {
			theme.UpdateSystemTheme(
				styles.Terminal.Background,
				styles.Terminal.BackgroundIsDark,
			)
			return dialog.ThemeSelectedMsg{
				ThemeName: theme.CurrentThemeName(),
			}
		}
	case modal.CloseModalMsg:
		var cmd tea.Cmd
		if a.modal != nil {
			cmd = a.modal.Close()
		}
		a.modal = nil
		return a, cmd
	case commands.ExecuteCommandMsg:
		updated, cmd := a.executeCommand(commands.Command(msg))
		return updated, cmd
	case commands.ExecuteCommandsMsg:
		for _, command := range msg {
			updated, cmd := a.executeCommand(command)
			if cmd != nil {
				return updated, cmd
			}
		}
	case app.SendMsg:
		a.showCompletionDialog = false
		cmd := a.app.SendChatMessage(context.Background(), msg.Text, msg.Attachments)
		cmds = append(cmds, cmd)
	case dialog.CompletionDialogCloseMsg:
		a.showCompletionDialog = false
	case opencode.EventListResponseEventInstallationUpdated:
		return a, toast.NewSuccessToast(
			"opencode updated to "+msg.Properties.Version+", restart to apply.",
			toast.WithTitle("New version installed"),
		)
	case opencode.EventListResponseEventSessionDeleted:
		if a.app.Session != nil && msg.Properties.Info.ID == a.app.Session.ID {
			a.app.Session = &opencode.Session{}
			a.app.Messages = []opencode.Message{}
		}
		return a, toast.NewSuccessToast("Session deleted successfully")
	case opencode.EventListResponseEventSessionUpdated:
		if msg.Properties.Info.ID == a.app.Session.ID {
			a.app.Session = &msg.Properties.Info
		}
	case opencode.EventListResponseEventMessageUpdated:
		if msg.Properties.Info.Metadata.SessionID == a.app.Session.ID {
			exists := false
			optimisticReplaced := false

			// First check if this is replacing an optimistic message
			if msg.Properties.Info.Role == opencode.MessageRoleUser {
				// Look for optimistic messages to replace
				for i, m := range a.app.Messages {
					if strings.HasPrefix(m.ID, "optimistic-") && m.Role == opencode.MessageRoleUser {
						// Replace the optimistic message with the real one
						a.app.Messages[i] = msg.Properties.Info
						exists = true
						optimisticReplaced = true
						break
					}
				}
			}

			// If not replacing optimistic, check for existing message with same ID
			if !optimisticReplaced {
				for i, m := range a.app.Messages {
					if m.ID == msg.Properties.Info.ID {
						a.app.Messages[i] = msg.Properties.Info
						exists = true
						break
					}
				}
			}

			if !exists {
				a.app.Messages = append(a.app.Messages, msg.Properties.Info)
			}
		}
	case opencode.EventListResponseEventSessionError:
		switch err := msg.Properties.Error.AsUnion().(type) {
		case nil:
		case opencode.ProviderAuthError:
			slog.Error("Failed to authenticate with provider", "error", err.Data.Message)
			return a, toast.NewErrorToast("Provider error: " + err.Data.Message)
		case opencode.UnknownError:
			slog.Error("Server error", "name", err.Name, "message", err.Data.Message)
			return a, toast.NewErrorToast(err.Data.Message, toast.WithTitle(string(err.Name)))
		}
	case tea.WindowSizeMsg:
		msg.Height -= 2 // Make space for the status bar
		a.width, a.height = msg.Width, msg.Height
		layout.Current = &layout.LayoutInfo{
			Viewport: layout.Dimensions{
				Width:  a.width,
				Height: a.height,
			},
			Container: layout.Dimensions{
				Width: min(a.width, 80),
			},
		}
		// Update child component sizes
		messagesHeight := a.height - 6 // Leave room for editor and status bar
		a.messages.SetSize(a.width, messagesHeight)
		a.editor.SetSize(min(a.width, 80), 5)
	case app.SessionSelectedMsg:
		messages, err := a.app.ListMessages(context.Background(), msg.ID)
		if err != nil {
			slog.Error("Failed to list messages", "error", err)
			return a, toast.NewErrorToast("Failed to open session")
		}
		a.app.Session = msg
		a.app.Messages = messages
	case app.ModelSelectedMsg:
		a.app.Provider = &msg.Provider
		a.app.Model = &msg.Model
		a.app.State.Provider = msg.Provider.ID
		a.app.State.Model = msg.Model.ID
		a.app.SaveState()
	case dialog.ThemeSelectedMsg:
		a.app.State.Theme = msg.ThemeName
		a.app.SaveState()
	case toast.ShowToastMsg:
		tm, cmd := a.toastManager.Update(msg)
		a.toastManager = tm
		cmds = append(cmds, cmd)
	case toast.DismissToastMsg:
		tm, cmd := a.toastManager.Update(msg)
		a.toastManager = tm
		cmds = append(cmds, cmd)
	case InterruptDebounceTimeoutMsg:
		// Reset interrupt key state after timeout
		a.interruptKeyState = InterruptKeyIdle
		a.editor.SetInterruptKeyInDebounce(false)
	}

	// update status bar
	s, cmd := a.status.Update(msg)
	cmds = append(cmds, cmd)
	a.status = s.(status.StatusComponent)

	// update editor
	u, cmd := a.editor.Update(msg)
	a.editor = u.(chat.EditorComponent)
	cmds = append(cmds, cmd)

	// update messages
	u, cmd = a.messages.Update(msg)
	a.messages = u.(chat.MessagesComponent)
	cmds = append(cmds, cmd)

	// update modal
	if a.modal != nil {
		u, cmd := a.modal.Update(msg)
		a.modal = u.(layout.Modal)
		cmds = append(cmds, cmd)
	}

	if a.showCompletionDialog {
		u, cmd := a.completions.Update(msg)
		a.completions = u.(dialog.CompletionDialog)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a appModel) View() string {
	messagesView := a.messages.View()
	editorView := a.editor.View()

	editorHeight := lipgloss.Height(editorView)
	if editorHeight < 5 {
		editorHeight = 5
	}

	t := theme.CurrentTheme()
	centeredEditorView := lipgloss.PlaceHorizontal(
		a.width,
		lipgloss.Center,
		editorView,
		styles.WhitespaceStyle(t.Background()),
	)

	mainLayout := layout.Render(
		layout.FlexOptions{
			Direction: layout.Column,
			Width:     a.width,
			Height:    a.height - 1, // Leave room for status bar
		},
		layout.FlexItem{
			View: messagesView,
			Grow: true,
		},
		layout.FlexItem{
			View:      centeredEditorView,
			FixedSize: editorHeight,
		},
	)

	if a.editor.Lines() > 1 {
		editorWidth := min(a.width, 80)
		editorX := (a.width - editorWidth) / 2
		editorY := a.height - editorHeight - 1 // Position from bottom, accounting for status bar

		mainLayout = layout.PlaceOverlay(
			editorX,
			editorY,
			a.editor.Content(),
			mainLayout,
		)
	}

	if a.showCompletionDialog {
		editorWidth := min(a.width, 80)
		editorX := (a.width - editorWidth) / 2
		a.completions.SetWidth(editorWidth)
		overlay := a.completions.View()
		overlayHeight := lipgloss.Height(overlay)
		editorY := a.height - editorHeight - 1

		mainLayout = layout.PlaceOverlay(
			editorX,
			editorY-overlayHeight,
			overlay,
			mainLayout,
		)
	}

	components := []string{
		mainLayout,
		a.status.View(),
	}
	appView := strings.Join(components, "\n")

	if a.modal != nil {
		appView = a.modal.Render(appView)
	}

	appView = a.toastManager.RenderOverlay(appView)

	if theme.CurrentThemeUsesAnsiColors() {
		appView = util.ConvertRGBToAnsi16Colors(appView)
	}

	return appView
}

func (a appModel) executeCommand(command commands.Command) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{
		util.CmdHandler(commands.CommandExecutedMsg(command)),
	}
	switch command.Name {
	case commands.AppHelpCommand:
		helpDialog := dialog.NewHelpDialog(a.app)
		a.modal = helpDialog
	case commands.EditorOpenCommand:
		if a.app.IsBusy() {
			// status.Warn("Agent is working, please wait...")
			return a, nil
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			return a, toast.NewErrorToast("No EDITOR set, can't open editor")
		}

		value := a.editor.Value()
		updated, cmd := a.editor.Clear()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)

		tmpfile, err := os.CreateTemp("", "msg_*.md")
		tmpfile.WriteString(value)
		if err != nil {
			slog.Error("Failed to create temp file", "error", err)
			return a, toast.NewErrorToast("Something went wrong, couldn't open editor")
		}
		tmpfile.Close()
		c := exec.Command(editor, tmpfile.Name()) //nolint:gosec
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		cmd = tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				slog.Error("Failed to open editor", "error", err)
				return nil
			}
			content, err := os.ReadFile(tmpfile.Name())
			if err != nil {
				slog.Error("Failed to read file", "error", err)
				return nil
			}
			if len(content) == 0 {
				slog.Warn("Message is empty")
				return nil
			}
			os.Remove(tmpfile.Name())
			// attachments := m.attachments
			// m.attachments = nil
			return app.SendMsg{
				Text:        string(content),
				Attachments: []app.Attachment{}, // attachments,
			}
		})
		cmds = append(cmds, cmd)
	case commands.SessionNewCommand:
		if a.app.Session.ID == "" {
			return a, nil
		}
		a.app.Session = &opencode.Session{}
		a.app.Messages = []opencode.Message{}
		cmds = append(cmds, util.CmdHandler(app.SessionClearedMsg{}))
	case commands.SessionListCommand:
		sessionDialog := dialog.NewSessionDialog(a.app)
		a.modal = sessionDialog
	case commands.SessionShareCommand:
		if a.app.Session.ID == "" {
			return a, nil
		}
		response, err := a.app.Client.Session.Share(context.Background(), a.app.Session.ID)
		if err != nil {
			slog.Error("Failed to share session", "error", err)
			return a, toast.NewErrorToast("Failed to share session")
		}
		shareUrl := response.Share.URL
		cmds = append(cmds, tea.SetClipboard(shareUrl))
		cmds = append(cmds, toast.NewSuccessToast("Share URL copied to clipboard!"))
	case commands.SessionInterruptCommand:
		if a.app.Session.ID == "" {
			return a, nil
		}
		a.app.Cancel(context.Background(), a.app.Session.ID)
		return a, nil
	case commands.SessionCompactCommand:
		if a.app.Session.ID == "" {
			return a, nil
		}
		// TODO: block until compaction is complete
		a.app.CompactSession(context.Background())
	case commands.ToolDetailsCommand:
		message := "Tool details are now visible"
		if a.messages.ToolDetailsVisible() {
			message = "Tool details are now hidden"
		}
		cmds = append(cmds, util.CmdHandler(chat.ToggleToolDetailsMsg{}))
		cmds = append(cmds, toast.NewInfoToast(message))
	case commands.ModelListCommand:
		modelDialog := dialog.NewModelDialog(a.app)
		a.modal = modelDialog
	case commands.ThemeListCommand:
		themeDialog := dialog.NewThemeDialog()
		a.modal = themeDialog
	case commands.ProjectInitCommand:
		cmds = append(cmds, a.app.InitializeProject(context.Background()))
	case commands.InputClearCommand:
		if a.editor.Value() == "" {
			return a, nil
		}
		updated, cmd := a.editor.Clear()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.InputPasteCommand:
		updated, cmd := a.editor.Paste()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.InputSubmitCommand:
		updated, cmd := a.editor.Submit()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.InputNewlineCommand:
		updated, cmd := a.editor.Newline()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.HistoryPreviousCommand:
		if a.showCompletionDialog {
			return a, nil
		}
		updated, cmd := a.editor.Previous()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.HistoryNextCommand:
		if a.showCompletionDialog {
			return a, nil
		}
		updated, cmd := a.editor.Next()
		a.editor = updated.(chat.EditorComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesFirstCommand:
		updated, cmd := a.messages.First()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesLastCommand:
		updated, cmd := a.messages.Last()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesPageUpCommand:
		updated, cmd := a.messages.PageUp()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesPageDownCommand:
		updated, cmd := a.messages.PageDown()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesHalfPageUpCommand:
		updated, cmd := a.messages.HalfPageUp()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.MessagesHalfPageDownCommand:
		updated, cmd := a.messages.HalfPageDown()
		a.messages = updated.(chat.MessagesComponent)
		cmds = append(cmds, cmd)
	case commands.AppExitCommand:
		return a, tea.Quit
	}
	return a, tea.Batch(cmds...)
}

func (a appModel) updateCompletions(msg tea.Msg) (tea.Model, tea.Cmd) {
	currentInput := a.editor.Value()
	if currentInput != "" {
		provider := a.completionManager.GetProvider(currentInput)
		a.completions.SetProvider(provider)
	}
	return a.completions.Update(msg)
}

func NewModel(app *app.App) tea.Model {
	completionManager := completions.NewCompletionManager(app)
	initialProvider := completionManager.DefaultProvider()

	messages := chat.NewMessagesComponent(app)
	editor := chat.NewEditorComponent(app)
	completions := dialog.NewCompletionDialogComponent(initialProvider)

	var leaderBinding *key.Binding
	if app.Config.Keybinds.Leader != "" {
		binding := key.NewBinding(key.WithKeys(app.Config.Keybinds.Leader))
		leaderBinding = &binding
	}

	model := &appModel{
		status:               status.NewStatusCmp(app),
		app:                  app,
		editor:               editor,
		messages:             messages,
		completions:          completions,
		completionManager:    completionManager,
		leaderBinding:        leaderBinding,
		isLeaderSequence:     false,
		showCompletionDialog: false,
		toastManager:         toast.NewToastManager(),
		interruptKeyState:    InterruptKeyIdle,
	}

	return model
}
