package lib

import (
	"encoding/json"
	"fmt"
)

// Adapted from
// https://gist.githubusercontent.com/MightyPork/6da26e382a7ad91b5496ee55fdc73db2/raw/e91b2eca00fdf3d8b51a4dddc658913d2baa40e0/usb_hid_keys.h

// Modifier masks - used for the first byte in the HID report.
// NOTE: The second byte in the report is reserved, 0x00.
const (
	KEY_MOD_LCTRL  = 0x01
	KEY_MOD_LSHIFT = 0x02
	KEY_MOD_LALT   = 0x04
	KEY_MOD_LMETA  = 0x08
	KEY_MOD_RCTRL  = 0x10
	KEY_MOD_RSHIFT = 0x20
	KEY_MOD_RALT   = 0x40
	KEY_MOD_RMETA  = 0x80
)

// Scan codes - last N slots in the HID report (usually 6).
// 0x00 if no key pressed.
// If more than N keys are pressed, the HID reports KEY_ERR_OVF in all slots.
const (
	KEY_NONE               = 0x00 // No key pressed
	KEY_ERR_OVF            = 0x01 // Keyboard Error Roll Over - used for all slots if too many keys are pressed ("Phantom key")
	KEY_A                  = 0x04 // Keyboard a and A
	KEY_B                  = 0x05 // Keyboard b and B
	KEY_C                  = 0x06 // Keyboard c and C
	KEY_D                  = 0x07 // Keyboard d and D
	KEY_E                  = 0x08 // Keyboard e and E
	KEY_F                  = 0x09 // Keyboard f and F
	KEY_G                  = 0x0A // Keyboard g and G
	KEY_H                  = 0x0B // Keyboard h and H
	KEY_I                  = 0x0C // Keyboard i and I
	KEY_J                  = 0x0D // Keyboard j and J
	KEY_K                  = 0x0E // Keyboard k and K
	KEY_L                  = 0x0F // Keyboard l and L
	KEY_M                  = 0x10 // Keyboard m and M
	KEY_N                  = 0x11 // Keyboard n and N
	KEY_O                  = 0x12 // Keyboard o and O
	KEY_P                  = 0x13 // Keyboard p and P
	KEY_Q                  = 0x14 // Keyboard q and Q
	KEY_R                  = 0x15 // Keyboard r and R
	KEY_S                  = 0x16 // Keyboard s and S
	KEY_T                  = 0x17 // Keyboard t and T
	KEY_U                  = 0x18 // Keyboard u and U
	KEY_V                  = 0x19 // Keyboard v and V
	KEY_W                  = 0x1A // Keyboard w and W
	KEY_X                  = 0x1B // Keyboard x and X
	KEY_Y                  = 0x1C // Keyboard y and Y
	KEY_Z                  = 0x1D // Keyboard z and Z
	KEY_1                  = 0x1E // Keyboard 1 and !
	KEY_2                  = 0x1F // Keyboard 2 and @
	KEY_3                  = 0x20 // Keyboard 3 and #
	KEY_4                  = 0x21 // Keyboard 4 and $
	KEY_5                  = 0x22 // Keyboard 5 and %
	KEY_6                  = 0x23 // Keyboard 6 and ^
	KEY_7                  = 0x24 // Keyboard 7 and &
	KEY_8                  = 0x25 // Keyboard 8 and *
	KEY_9                  = 0x26 // Keyboard 9 and (
	KEY_0                  = 0x27 // Keyboard 0 and )
	KEY_ENTER              = 0x28 // Keyboard Return (ENTER)
	KEY_ESC                = 0x29 // Keyboard ESCAPE
	KEY_BACKSPACE          = 0x2A // Keyboard DELETE (Backspace)
	KEY_TAB                = 0x2B // Keyboard Tab
	KEY_SPACE              = 0x2C // Keyboard Spacebar
	KEY_MINUS              = 0x2D // Keyboard - and _
	KEY_EQUAL              = 0x2E // Keyboard = and +
	KEY_LEFTBRACE          = 0x2F // Keyboard [ and {
	KEY_RIGHTBRACE         = 0x30 // Keyboard ] and }
	KEY_BACKSLASH          = 0x31 // Keyboard \ and |
	KEY_HASHTILDE          = 0x32 // Keyboard Non-US # and ~
	KEY_SEMICOLON          = 0x33 // Keyboard ; and :
	KEY_APOSTROPHE         = 0x34 // Keyboard ' and "
	KEY_GRAVE              = 0x35 // Keyboard ` and ~
	KEY_COMMA              = 0x36 // Keyboard , and <
	KEY_DOT                = 0x37 // Keyboard . and >
	KEY_SLASH              = 0x38 // Keyboard / and ?
	KEY_CAPSLOCK           = 0x39 // Keyboard Caps Lock
	KEY_F1                 = 0x3A // Keyboard F1
	KEY_F2                 = 0x3B // Keyboard F2
	KEY_F3                 = 0x3C // Keyboard F3
	KEY_F4                 = 0x3D // Keyboard F4
	KEY_F5                 = 0x3E // Keyboard F5
	KEY_F6                 = 0x3F // Keyboard F6
	KEY_F7                 = 0x40 // Keyboard F7
	KEY_F8                 = 0x41 // Keyboard F8
	KEY_F9                 = 0x42 // Keyboard F9
	KEY_F10                = 0x43 // Keyboard F10
	KEY_F11                = 0x44 // Keyboard F11
	KEY_F12                = 0x45 // Keyboard F12
	KEY_SYSRQ              = 0x46 // Keyboard Print Screen
	KEY_SCROLLLOCK         = 0x47 // Keyboard Scroll Lock
	KEY_PAUSE              = 0x48 // Keyboard Pause
	KEY_INSERT             = 0x49 // Keyboard Insert
	KEY_HOME               = 0x4A // Keyboard Home
	KEY_PAGEUP             = 0x4B // Keyboard Page Up
	KEY_DELETE             = 0x4C // Keyboard Delete Forward
	KEY_END                = 0x4D // Keyboard End
	KEY_PAGEDOWN           = 0x4E // Keyboard Page Down
	KEY_RIGHT              = 0x4F // Keyboard Right Arrow
	KEY_LEFT               = 0x50 // Keyboard Left Arrow
	KEY_DOWN               = 0x51 // Keyboard Down Arrow
	KEY_UP                 = 0x52 // Keyboard Up Arrow
	KEY_NUMLOCK            = 0x53 // Keyboard Num Lock and Clear
	KEY_KPSLASH            = 0x54 // Keypad /
	KEY_KPASTERISK         = 0x55 // Keypad *
	KEY_KPMINUS            = 0x56 // Keypad -
	KEY_KPPLUS             = 0x57 // Keypad +
	KEY_KPENTER            = 0x58 // Keypad ENTER
	KEY_KP1                = 0x59 // Keypad 1 and End
	KEY_KP2                = 0x5A // Keypad 2 and Down Arrow
	KEY_KP3                = 0x5B // Keypad 3 and PageDn
	KEY_KP4                = 0x5C // Keypad 4 and Left Arrow
	KEY_KP5                = 0x5D // Keypad 5
	KEY_KP6                = 0x5E // Keypad 6 and Right Arrow
	KEY_KP7                = 0x5F // Keypad 7 and Home
	KEY_KP8                = 0x60 // Keypad 8 and Up Arrow
	KEY_KP9                = 0x61 // Keypad 9 and Page Up
	KEY_KP0                = 0x62 // Keypad 0 and Insert
	KEY_KPDOT              = 0x63 // Keypad . and Delete
	KEY_102ND              = 0x64 // Keyboard Non-US \ and |
	KEY_COMPOSE            = 0x65 // Keyboard Application
	KEY_POWER              = 0x66 // Keyboard Power
	KEY_KPEQUAL            = 0x67 // Keypad =
	KEY_F13                = 0x68 // Keyboard F13
	KEY_F14                = 0x69 // Keyboard F14
	KEY_F15                = 0x6A // Keyboard F15
	KEY_F16                = 0x6B // Keyboard F16
	KEY_F17                = 0x6C // Keyboard F17
	KEY_F18                = 0x6D // Keyboard F18
	KEY_F19                = 0x6E // Keyboard F19
	KEY_F20                = 0x6F // Keyboard F20
	KEY_F21                = 0x70 // Keyboard F21
	KEY_F22                = 0x71 // Keyboard F22
	KEY_F23                = 0x72 // Keyboard F23
	KEY_F24                = 0x73 // Keyboard F24
	KEY_OPEN               = 0x74 // Keyboard Execute
	KEY_HELP               = 0x75 // Keyboard Help
	KEY_PROPS              = 0x76 // Keyboard Menu
	KEY_FRONT              = 0x77 // Keyboard Select
	KEY_STOP               = 0x78 // Keyboard Stop
	KEY_AGAIN              = 0x79 // Keyboard Again
	KEY_UNDO               = 0x7A // Keyboard Undo
	KEY_CUT                = 0x7B // Keyboard Cut
	KEY_COPY               = 0x7C // Keyboard Copy
	KEY_PASTE              = 0x7D // Keyboard Paste
	KEY_FIND               = 0x7E // Keyboard Find
	KEY_MUTE               = 0x7F // Keyboard Mute
	KEY_VOLUMEUP           = 0x80 // Keyboard Volume Up
	KEY_VOLUMEDOWN         = 0x81 // Keyboard Volume Down
	KEY_KPCOMMA            = 0x85 // Keypad Comma
	KEY_RO                 = 0x87 // Keyboard International1
	KEY_KATAKANAHIRAGANA   = 0x88 // Keyboard International2
	KEY_YEN                = 0x89 // Keyboard International3
	KEY_HENKAN             = 0x8A // Keyboard International4
	KEY_MUHENKAN           = 0x8B // Keyboard International5
	KEY_KPJPCOMMA          = 0x8C // Keyboard International6
	KEY_HANGEUL            = 0x90 // Keyboard LANG1
	KEY_HANJA              = 0x91 // Keyboard LANG2
	KEY_KATAKANA           = 0x92 // Keyboard LANG3
	KEY_HIRAGANA           = 0x93 // Keyboard LANG4
	KEY_ZENKAKUHANKAKU     = 0x94 // Keyboard LANG5
	KEY_KPLEFTPAREN        = 0xB6 // Keypad (
	KEY_KPRIGHTPAREN       = 0xB7 // Keypad )
	KEY_LEFTCTRL           = 0xE0 // Keyboard Left Control
	KEY_LEFTSHIFT          = 0xE1 // Keyboard Left Shift
	KEY_LEFTALT            = 0xE2 // Keyboard Left Alt
	KEY_LEFTMETA           = 0xE3 // Keyboard Left GUI
	KEY_RIGHTCTRL          = 0xE4 // Keyboard Right Control
	KEY_RIGHTSHIFT         = 0xE5 // Keyboard Right Shift
	KEY_RIGHTALT           = 0xE6 // Keyboard Right Alt
	KEY_RIGHTMETA          = 0xE7 // Keyboard Right GUI
	KEY_MEDIA_PLAYPAUSE    = 0xE8
	KEY_MEDIA_STOPCD       = 0xE9
	KEY_MEDIA_PREVIOUSSONG = 0xEA
	KEY_MEDIA_NEXTSONG     = 0xEB
	KEY_MEDIA_EJECTCD      = 0xEC
	KEY_MEDIA_VOLUMEUP     = 0xED
	KEY_MEDIA_VOLUMEDOWN   = 0xEE
	KEY_MEDIA_MUTE         = 0xEF
	KEY_MEDIA_WWW          = 0xF0
	KEY_MEDIA_BACK         = 0xF1
	KEY_MEDIA_FORWARD      = 0xF2
	KEY_MEDIA_STOP         = 0xF3
	KEY_MEDIA_FIND         = 0xF4
	KEY_MEDIA_SCROLLUP     = 0xF5
	KEY_MEDIA_SCROLLDOWN   = 0xF6
	KEY_MEDIA_EDIT         = 0xF7
	KEY_MEDIA_SLEEP        = 0xF8
	KEY_MEDIA_COFFEE       = 0xF9
	KEY_MEDIA_REFRESH      = 0xFA
	KEY_MEDIA_CALC         = 0xFB
)

var KeyNames = map[int]string{
	KEY_NONE:       "None",
	KEY_ERR_OVF:    "Error Overflow",
	KEY_A:          "A",
	KEY_B:          "B",
	KEY_C:          "C",
	KEY_D:          "D",
	KEY_E:          "E",
	KEY_F:          "F",
	KEY_G:          "G",
	KEY_H:          "H",
	KEY_I:          "I",
	KEY_J:          "J",
	KEY_K:          "K",
	KEY_L:          "L",
	KEY_M:          "M",
	KEY_N:          "N",
	KEY_O:          "O",
	KEY_P:          "P",
	KEY_Q:          "Q",
	KEY_R:          "R",
	KEY_S:          "S",
	KEY_T:          "T",
	KEY_U:          "U",
	KEY_V:          "V",
	KEY_W:          "W",
	KEY_X:          "X",
	KEY_Y:          "Y",
	KEY_Z:          "Z",
	KEY_1:          "1",
	KEY_2:          "2",
	KEY_3:          "3",
	KEY_4:          "4",
	KEY_5:          "5",
	KEY_6:          "6",
	KEY_7:          "7",
	KEY_8:          "8",
	KEY_9:          "9",
	KEY_0:          "0",
	KEY_ENTER:      "Enter",
	KEY_ESC:        "Escape",
	KEY_BACKSPACE:  "Backspace",
	KEY_TAB:        "Tab",
	KEY_SPACE:      "Space",
	KEY_MINUS:      "-",
	KEY_EQUAL:      "=",
	KEY_LEFTBRACE:  "[",
	KEY_RIGHTBRACE: "]",
	KEY_BACKSLASH:  "\\",
	KEY_HASHTILDE:  "#",
	KEY_SEMICOLON:  ";",
	KEY_APOSTROPHE: "'",
	KEY_GRAVE:      "`",
	KEY_COMMA:      ",",
	KEY_DOT:        ".",
	KEY_SLASH:      "/",
	KEY_CAPSLOCK:   "Caps Lock",
	KEY_F1:         "F1",
	KEY_F2:         "F2",
	KEY_F3:         "F3",
	KEY_F4:         "F4",
	KEY_F5:         "F5",
	KEY_F6:         "F6",
	KEY_F7:         "F7",
	KEY_F8:         "F8",
	KEY_F9:         "F9",
	KEY_F10:        "F10",
	KEY_F11:        "F11",
	KEY_F12:        "F12",
	KEY_SYSRQ:      "Print Screen",
	KEY_SCROLLLOCK: "Scroll Lock",
	KEY_PAUSE:      "Pause",
	KEY_INSERT:     "Insert",
	KEY_HOME:       "Home",
	KEY_PAGEUP:     "Page Up",
	KEY_DELETE:     "Delete",
	KEY_END:        "End",
	KEY_PAGEDOWN:   "Page Down",
	KEY_RIGHT:      "Right Arrow",
	KEY_LEFT:       "Left Arrow",
	KEY_DOWN:       "Down Arrow",
	KEY_UP:         "Up Arrow",
	KEY_NUMLOCK:    "Num Lock",
	KEY_KPSLASH:    "Keypad /",
	KEY_KPASTERISK: "Keypad *",
	KEY_KPMINUS:    "Keypad -",
	KEY_KPPLUS:     "Keypad +",
	KEY_KPENTER:    "Keypad Enter",
	KEY_KP1:        "Keypad 1",
	KEY_KP2:        "Keypad 2",
	KEY_KP3:        "Keypad 3",
	KEY_KP4:        "Keypad 4",
	KEY_KP5:        "Keypad 5",
	KEY_KP6:        "Keypad 6",
	KEY_KP7:        "Keypad 7",
	KEY_KP8:        "Keypad 8",
	KEY_KP9:        "Keypad 9",
	KEY_KP0:        "Keypad 0"}

var ModifierNames = map[byte]string{
	KEY_MOD_LCTRL:  "Left Control",
	KEY_MOD_LSHIFT: "Left Shift",
	KEY_MOD_LALT:   "Left Alt",
	KEY_MOD_LMETA:  "Left Meta",
	KEY_MOD_RCTRL:  "Right Control",
	KEY_MOD_RSHIFT: "Right Shift",
	KEY_MOD_RALT:   "Right Alt",
	KEY_MOD_RMETA:  "Right Meta",
}

func KeyName(keyCode int) string {
	if name, found := KeyNames[keyCode]; found {
		return name
	}
	return fmt.Sprintf("Unknown Key (0x%X)", keyCode)
}

func ModifierName(modCode byte) string {
	if name, found := ModifierNames[modCode]; found {
		return name
	}
	return fmt.Sprintf("Unknown Modifier (0x%X)", modCode)
}

// ReadableHIDReport represents a readable HID report.
type ReadableHIDReport struct {
	Modifiers []string `json:"modifiers"`
	Keys      []string `json:"keys"`
}

func CreateReadableHIDReport(report []byte) (*ReadableHIDReport, error) {
	if len(report) < 8 {
		return nil, fmt.Errorf("HID report must be at least 8 bytes long")
	}

	modifiers := report[0]
	keys := report[2:8]

	modifierNames := []string{}
	for i := 0; i < 8; i++ {
		bit := byte(1 << i)
		if modifiers&bit != 0 {
			modifierNames = append(modifierNames, ModifierName(bit))
		}
	}

	keyNames := []string{}
	for _, key := range keys {
		if key != 0 {
			keyNames = append(keyNames, KeyName(int(key)))
		}
	}

	hidReport := ReadableHIDReport{
		Modifiers: modifierNames,
		Keys:      keyNames,
	}
	return &hidReport, nil
}

func (hid *ReadableHIDReport) ToNativeHIDReport() NativeHIDReport {
	modifiers := byte(0)
	for _, mod := range hid.Modifiers {
		for i := 0; i < 8; i++ {
			bit := byte(1 << i)
			if ModifierName(bit) == mod {
				modifiers |= bit
			}
		}
	}

	keys := []int{}
	for _, key := range hid.Keys {
		for code := range KeyNames {
			if KeyName(code) == key {
				keys = append(keys, int(code))
			}
		}
	}

	return NativeHIDReport{
		Modifiers: modifiers,
		Keys:      keys,
	}
}

func (readableReport *ReadableHIDReport) ToJSON() (string, error) {
	jsonData, err := json.Marshal(readableReport)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// NativeHIDReport represents a HID report using native types.
type NativeHIDReport struct {
	Modifiers byte  `json:"modifiers"`
	Keys      []int `json:"keys"` //use [] int for better JSON serialization
}

func CreateNativeHIDReport(report []byte) (*NativeHIDReport, error) {
	if len(report) < 8 {
		return nil, fmt.Errorf("HID report must be at least 8 bytes long")
	}

	nativeReport := &NativeHIDReport{
		Modifiers: report[0],
		Keys:      byteSliceToIntSlice(report[2:8]),
	}
	return nativeReport, nil
}

func byteSliceToIntSlice(b []byte) []int {
	intSlice := make([]int, len(b)) // Create a new slice of integers with the same length as the byte slice
	for i, v := range b {
		intSlice[i] = int(v) // Convert each byte to an int
	}
	return intSlice
}

func (nativeReport *NativeHIDReport) ToJSON() (string, error) {
	jsonData, err := json.Marshal(nativeReport)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func (native *NativeHIDReport) ToReadableHIDReport() ReadableHIDReport {
	modifierNames := []string{}
	for i := 0; i < 8; i++ {
		bit := byte(1 << i)
		if native.Modifiers&bit != 0 {
			modifierNames = append(modifierNames, ModifierName(bit))
		}
	}

	keyNames := []string{}
	for _, key := range native.Keys {
		if key != 0 {
			keyNames = append(keyNames, KeyName(key))
		}
	}

	return ReadableHIDReport{
		Modifiers: modifierNames,
		Keys:      keyNames,
	}
}
