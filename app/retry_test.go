package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		Intervals:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
		SleepFunc:  func(d time.Duration) {},
	}

	callCount := 0
	err := Retry(config, "test operation", func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		Intervals:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
		SleepFunc:  func(d time.Duration) {},
	}

	callCount := 0
	err := Retry(config, "test operation", func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		Intervals:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
		SleepFunc:  func(d time.Duration) {},
	}

	callCount := 0
	expectedErr := errors.New("persistent error")
	err := Retry(config, "test operation", func() error {
		callCount++
		return expectedErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 初回 + 3リトライ = 4回
	if callCount != 4 {
		t.Errorf("expected 4 calls (1 initial + 3 retries), got %d", callCount)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error to contain original error")
	}
}

func TestRetry_ExponentialBackoffIntervals(t *testing.T) {
	var sleepDurations []time.Duration
	config := RetryConfig{
		MaxRetries: 3,
		Intervals:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
		SleepFunc: func(d time.Duration) {
			sleepDurations = append(sleepDurations, d)
		},
	}

	_ = Retry(config, "test operation", func() error {
		return errors.New("always fail")
	})

	expectedIntervals := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	if len(sleepDurations) != len(expectedIntervals) {
		t.Fatalf("expected %d sleep calls, got %d", len(expectedIntervals), len(sleepDurations))
	}
	for i, expected := range expectedIntervals {
		if sleepDurations[i] != expected {
			t.Errorf("sleep[%d]: expected %v, got %v", i, expected, sleepDurations[i])
		}
	}
}

func TestRetry_NoSleepOnSuccess(t *testing.T) {
	sleepCalled := false
	config := RetryConfig{
		MaxRetries: 3,
		Intervals:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
		SleepFunc: func(d time.Duration) {
			sleepCalled = true
		},
	}

	err := Retry(config, "test operation", func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sleepCalled {
		t.Error("sleep should not be called on success")
	}
}

func TestReadImageFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jpg")
	content := []byte("fake image data")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	data, err := ReadImageFile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(data))
	}
}

func TestReadImageFile_NonExistentFile(t *testing.T) {
	_, err := ReadImageFile("/nonexistent/path/image.jpg")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestReadImageFile_Directory(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadImageFile(dir)
	if err == nil {
		t.Fatal("expected error for directory, got nil")
	}
}

func TestReadImageFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jpg")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := ReadImageFile(path)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
	}

	expectedIntervals := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	if len(config.Intervals) != len(expectedIntervals) {
		t.Fatalf("expected %d intervals, got %d", len(expectedIntervals), len(config.Intervals))
	}
	for i, expected := range expectedIntervals {
		if config.Intervals[i] != expected {
			t.Errorf("interval[%d]: expected %v, got %v", i, expected, config.Intervals[i])
		}
	}

	if config.SleepFunc == nil {
		t.Error("expected SleepFunc to be set")
	}
}
