package main

import (
	"testing"
	"time"
)

// Legacy macOS key codes for backward compatibility with tests
const (
	KeyW     int64 = 13
	KeyA     int64 = 0
	KeyS     int64 = 1
	KeyD     int64 = 2
	KeyQ     int64 = 12
	KeyE     int64 = 14
	KeyZ     int64 = 6
	KeyX     int64 = 7
	KeyR     int64 = 15
	KeyF     int64 = 3
	KeySpace int64 = 49
)

func TestNewMouseController(t *testing.T) {
	mc := NewMouseController()
	if mc == nil {
		t.Fatal("NewMouseController returned nil")
	}
	if mc.active {
		t.Error("New controller should not be active")
	}
}

func TestToggle(t *testing.T) {
	mc := NewMouseController()

	// Initially inactive
	if mc.IsActive() {
		t.Error("Controller should start inactive")
	}

	// Toggle on
	mc.Toggle()
	if !mc.IsActive() {
		t.Error("Controller should be active after toggle")
	}

	// Toggle off
	mc.Toggle()
	if mc.IsActive() {
		t.Error("Controller should be inactive after second toggle")
	}
}

func TestToggleResetsKeyState(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle() // Activate

	// Set some key states
	mc.HandleKeyDown(KeyW)
	mc.HandleKeyDown(KeyA)

	// Toggle off should reset states
	mc.Toggle()

	// Verify keys are reset
	mc.mu.Lock()
	if mc.keyW || mc.keyA || mc.keyS || mc.keyD {
		t.Error("Toggle off should reset all key states")
	}
	mc.mu.Unlock()
}

func TestHandleKeyDownWhenInactive(t *testing.T) {
	mc := NewMouseController()

	// Should return false when inactive
	if mc.HandleKeyDown(KeyW) {
		t.Error("HandleKeyDown should return false when inactive")
	}
}

func TestHandleKeyDownWhenActive(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle() // Activate

	tests := []struct {
		keycode int64
		name    string
	}{
		{KeyW, "W"},
		{KeyA, "A"},
		{KeyS, "S"},
		{KeyD, "D"},
		{KeyQ, "Q"},
		{KeyE, "E"},
		{KeyZ, "Z"},
		{KeyX, "X"},
		{KeySpace, "Space"},
		{KeyR, "R"},
		{KeyF, "F"},
	}

	for _, tt := range tests {
		if !mc.HandleKeyDown(tt.keycode) {
			t.Errorf("HandleKeyDown(%s) should return true when active", tt.name)
		}
	}
}

func TestHandleKeyUpWhenInactive(t *testing.T) {
	mc := NewMouseController()

	if mc.HandleKeyUp(KeyW) {
		t.Error("HandleKeyUp should return false when inactive")
	}
}

func TestHandleKeyUpWhenActive(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle() // Activate

	// First press then release
	mc.HandleKeyDown(KeyW)
	if !mc.HandleKeyUp(KeyW) {
		t.Error("HandleKeyUp(W) should return true when active")
	}
}

func TestGetMovementWhenInactive(t *testing.T) {
	mc := NewMouseController()

	dx, dy := mc.GetMovement()
	if dx != 0 || dy != 0 {
		t.Errorf("GetMovement should return (0,0) when inactive, got (%f,%f)", dx, dy)
	}
}

func TestGetMovementNoKeys(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle() // Activate

	dx, dy := mc.GetMovement()
	if dx != 0 || dy != 0 {
		t.Errorf("GetMovement should return (0,0) with no keys pressed, got (%f,%f)", dx, dy)
	}
}

func TestGetMovementCardinalDirections(t *testing.T) {
	tests := []struct {
		name     string
		keycode  int64
		expectDx float64
		expectDy float64
	}{
		{"W - Up", KeyW, 0, -1},
		{"S - Down", KeyS, 0, 1},
		{"A - Left", KeyA, -1, 0},
		{"D - Right", KeyD, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMouseController()
			mc.Toggle()
			mc.HandleKeyDown(tt.keycode)

			dx, dy := mc.GetMovement()

			// At base speed, check direction
			if tt.expectDx < 0 && dx >= 0 {
				t.Errorf("Expected negative dx, got %f", dx)
			}
			if tt.expectDx > 0 && dx <= 0 {
				t.Errorf("Expected positive dx, got %f", dx)
			}
			if tt.expectDy < 0 && dy >= 0 {
				t.Errorf("Expected negative dy, got %f", dy)
			}
			if tt.expectDy > 0 && dy <= 0 {
				t.Errorf("Expected positive dy, got %f", dy)
			}
			if tt.expectDx == 0 && dx != 0 {
				t.Errorf("Expected dx=0, got %f", dx)
			}
			if tt.expectDy == 0 && dy != 0 {
				t.Errorf("Expected dy=0, got %f", dy)
			}
		})
	}
}

func TestGetMovementDiagonalDirections(t *testing.T) {
	tests := []struct {
		name     string
		keycode  int64
		expectDx float64
		expectDy float64
	}{
		{"Q - Up-Left", KeyQ, -1, -1},
		{"E - Up-Right", KeyE, 1, -1},
		{"Z - Down-Left", KeyZ, -1, 1},
		{"X - Down-Right", KeyX, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMouseController()
			mc.Toggle()
			mc.HandleKeyDown(tt.keycode)

			dx, dy := mc.GetMovement()

			// Check directions (diagonal keys produce ~0.707 factor)
			if tt.expectDx < 0 && dx >= 0 {
				t.Errorf("Expected negative dx, got %f", dx)
			}
			if tt.expectDx > 0 && dx <= 0 {
				t.Errorf("Expected positive dx, got %f", dx)
			}
			if tt.expectDy < 0 && dy >= 0 {
				t.Errorf("Expected negative dy, got %f", dy)
			}
			if tt.expectDy > 0 && dy <= 0 {
				t.Errorf("Expected positive dy, got %f", dy)
			}
		})
	}
}

func TestGetMovementDiagonalNormalization(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()

	// Press W and D together (diagonal via cardinal keys)
	mc.HandleKeyDown(KeyW)
	mc.HandleKeyDown(KeyD)

	dx, dy := mc.GetMovement()

	// Combined cardinal should be normalized (0.707 factor)
	// At base speed = 1.0, diagonal should be ~0.707 each direction
	expectedMagnitude := 0.707 * baseSpeed
	tolerance := 0.01

	if dx < expectedMagnitude-tolerance || dx > expectedMagnitude+tolerance {
		t.Errorf("Diagonal dx should be ~%f, got %f", expectedMagnitude, dx)
	}
	if dy > -expectedMagnitude+tolerance || dy < -expectedMagnitude-tolerance {
		t.Errorf("Diagonal dy should be ~%f, got %f", -expectedMagnitude, dy)
	}
}

func TestAcceleration(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()
	mc.HandleKeyDown(KeyD) // Move right

	// First call - should be at base speed
	dx1, _ := mc.GetMovement()

	// Wait and check acceleration
	time.Sleep(200 * time.Millisecond)
	dx2, _ := mc.GetMovement()

	if dx2 <= dx1 {
		t.Errorf("Speed should increase over time: initial=%f, after 200ms=%f", dx1, dx2)
	}
}

func TestAccelerationResetOnDirectionChange(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()

	// Move right for a bit
	mc.HandleKeyDown(KeyD)
	mc.GetMovement()
	time.Sleep(100 * time.Millisecond)
	dxBefore, _ := mc.GetMovement()

	// Change direction
	mc.HandleKeyUp(KeyD)
	mc.HandleKeyDown(KeyA)

	// Speed should reset to base
	dxAfter, _ := mc.GetMovement()

	// dxAfter should be negative (left) and close to base speed
	if dxAfter >= 0 {
		t.Error("Direction should have changed to left")
	}
	if -dxAfter > dxBefore {
		t.Errorf("Speed should reset on direction change: before=%f, after=%f", dxBefore, -dxAfter)
	}
}

func TestSpaceKeyLeftClick(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()

	// Press space
	mc.HandleKeyDown(KeySpace)

	mc.mu.Lock()
	leftDown := mc.leftDown
	mc.mu.Unlock()

	if !leftDown {
		t.Error("Space should set leftDown to true")
	}

	// Release space
	mc.HandleKeyUp(KeySpace)

	mc.mu.Lock()
	leftDown = mc.leftDown
	mc.mu.Unlock()

	if leftDown {
		t.Error("Releasing space should set leftDown to false")
	}
}

func TestUnknownKeyReturnsTrue(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()

	// Unknown key should return false (not handled)
	unknownKey := int64(999)
	if mc.HandleKeyDown(unknownKey) {
		t.Error("Unknown key should return false")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mc := NewMouseController()
	mc.Toggle()

	done := make(chan bool)

	// Goroutine pressing keys
	go func() {
		for i := 0; i < 100; i++ {
			mc.HandleKeyDown(KeyW)
			mc.HandleKeyUp(KeyW)
		}
		done <- true
	}()

	// Goroutine getting movement
	go func() {
		for i := 0; i < 100; i++ {
			mc.GetMovement()
		}
		done <- true
	}()

	// Goroutine toggling
	go func() {
		for i := 0; i < 10; i++ {
			mc.Toggle()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all
	<-done
	<-done
	<-done

	// If we get here without deadlock or panic, test passes
}
