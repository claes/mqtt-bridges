package lib

import (
	"testing"
)

func TestCreateReadableHIDReport(t *testing.T) {
	report := []byte{KEY_MOD_LCTRL | KEY_MOD_RSHIFT, 0x00, KEY_A, KEY_B, KEY_NONE, KEY_NONE, KEY_NONE, KEY_NONE}
	expectedModifiers := []string{"Left Control", "Right Shift"}
	expectedKeys := []string{"A", "B"}

	readable, err := CreateReadableHIDReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(readable.Modifiers) != len(expectedModifiers) {
		t.Errorf("expected %v modifiers, got %v", expectedModifiers, readable.Modifiers)
	}
	for i, mod := range expectedModifiers {
		if readable.Modifiers[i] != mod {
			t.Errorf("expected modifier %v, got %v", mod, readable.Modifiers[i])
		}
	}

	if len(readable.Keys) != len(expectedKeys) {
		t.Errorf("expected %v keys, got %v", expectedKeys, readable.Keys)
	}
	for i, key := range expectedKeys {
		if readable.Keys[i] != key {
			t.Errorf("expected key %v, got %v", key, readable.Keys[i])
		}
	}
}

func TestToNativeHIDReport(t *testing.T) {
	readable := ReadableHIDReport{
		Modifiers: []string{"Left Control", "Right Shift"},
		Keys:      []string{"A", "B"},
	}
	expectedModifiers := byte(KEY_MOD_LCTRL | KEY_MOD_RSHIFT)
	expectedKeys := []int{KEY_A, KEY_B}

	native := readable.ToNativeHIDReport()

	if native.Modifiers != expectedModifiers {
		t.Errorf("expected modifiers 0x%X, got 0x%X", expectedModifiers, native.Modifiers)
	}

	if len(native.Keys) != len(expectedKeys) {
		t.Errorf("expected %v keys, got %v", expectedKeys, native.Keys)
	}
	for i, key := range expectedKeys {
		if native.Keys[i] != key {
			t.Errorf("expected key %v, got %v", key, native.Keys[i])
		}
	}
}

func TestRoundTripReadableToNative(t *testing.T) {
	originalReadable := ReadableHIDReport{
		Modifiers: []string{"Left Control", "Right Shift"},
		Keys:      []string{"A", "B"},
	}

	native := originalReadable.ToNativeHIDReport()
	convertedReadable := native.ToReadableHIDReport()

	if len(originalReadable.Modifiers) != len(convertedReadable.Modifiers) {
		t.Errorf("expected %v modifiers, got %v", originalReadable.Modifiers, convertedReadable.Modifiers)
	}
	for i, mod := range originalReadable.Modifiers {
		if convertedReadable.Modifiers[i] != mod {
			t.Errorf("expected modifier %v, got %v", mod, convertedReadable.Modifiers[i])
		}
	}

	if len(originalReadable.Keys) != len(convertedReadable.Keys) {
		t.Errorf("expected %v keys, got %v", originalReadable.Keys, convertedReadable.Keys)
	}
	for i, key := range originalReadable.Keys {
		if convertedReadable.Keys[i] != key {
			t.Errorf("expected key %v, got %v", key, convertedReadable.Keys[i])
		}
	}
}

func TestRoundTripNativeToReadable(t *testing.T) {
	originalNative := NativeHIDReport{
		Modifiers: byte(KEY_MOD_LCTRL | KEY_MOD_RSHIFT),
		Keys:      []int{KEY_A, KEY_B},
	}

	readable := originalNative.ToReadableHIDReport()
	convertedNative := readable.ToNativeHIDReport()

	if originalNative.Modifiers != convertedNative.Modifiers {
		t.Errorf("expected modifiers 0x%X, got 0x%X", originalNative.Modifiers, convertedNative.Modifiers)
	}

	if len(originalNative.Keys) != len(convertedNative.Keys) {
		t.Errorf("expected %v keys, got %v", originalNative.Keys, convertedNative.Keys)
	}
	for i, key := range originalNative.Keys {
		if convertedNative.Keys[i] != key {
			t.Errorf("expected key %v, got %v", key, convertedNative.Keys[i])
		}
	}
}

func TestCreateNativeHIDReport(t *testing.T) {
	report := []byte{KEY_MOD_LCTRL | KEY_MOD_RSHIFT, 0x00, KEY_A, KEY_B, KEY_NONE, KEY_NONE, KEY_NONE, KEY_NONE}
	expectedModifiers := byte(KEY_MOD_LCTRL | KEY_MOD_RSHIFT)
	expectedKeys := []int{KEY_A, KEY_B, 0, 0, 0, 0}

	native, err := CreateNativeHIDReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if native.Modifiers != expectedModifiers {
		t.Errorf("expected modifiers 0x%X, got 0x%X", expectedModifiers, native.Modifiers)
	}

	if len(native.Keys) != len(expectedKeys) {
		t.Errorf("expected %v keys, got %v", expectedKeys, native.Keys)
	}
	for i, key := range expectedKeys {
		if native.Keys[i] != key {
			t.Errorf("expected key %v, got %v", key, native.Keys[i])
		}
	}
}
