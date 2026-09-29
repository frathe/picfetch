package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/frathe/picfetch/internal/locationtrial"
)

// Carbon virtual key positions used by the native Fyne keyboard path. Keypad
// plus/minus avoid the layout-dependent ANSI punctuation positions.
const (
	mapKeyL      = 0x25
	mapKeyEscape = 0x35
	mapKeyLeft   = 0x7b
	mapKeyRight  = 0x7c
	mapKeyPlus   = 0x45
	mapKeyMinus  = 0x4e
)

// Keep the evidence boundary aligned with responseTimeoutNS in native/capture.swift.
const responseTimeoutNS int64 = 3_000_000_000

func validNativeResponseTime(inputNS, visibleNS int64) bool {
	return inputNS > 0 && visibleNS > inputNS && visibleNS-inputNS < responseTimeoutNS
}

type nativeCommand struct {
	Kind  string `json:"kind"`
	Key   uint16 `json:"key"`
	Shift bool   `json:"shift"`
	Name  string `json:"name"`
}
type nativeObservation struct {
	Gesture
	Error        string `json:"error"`
	ClosedViewer bool   `json:"closed_viewer"`
}
type nativeDriver interface {
	State(context.Context) (locationtrial.State, error)
	Input(context.Context, nativeCommand) (nativeObservation, error)
}

// One schedule drives collection and validates retained witnesses, including
// gestures beyond the first forty during the sustained 30k browse.
func nativeGestureCommand(index int) nativeCommand {
	kind, key := "pan", uint16(mapKeyLeft)
	if index%2 == 1 {
		kind, key = "zoom", mapKeyPlus
	}
	if index%4 >= 2 {
		if kind == "pan" {
			key = mapKeyRight
		} else {
			key = mapKeyMinus
		}
	}
	return nativeCommand{Kind: kind, Key: key, Shift: kind == "pan", Name: fmt.Sprintf("gesture-%03d", index)}
}

func (c nativeCommand) matchesTransform(v *VisualTransform) bool {
	if !v.validFor(c.Kind) {
		return false
	}
	switch c.Kind {
	case "pan":
		return c.Shift && (c.Key == mapKeyLeft && v.DX > 0 || c.Key == mapKeyRight && v.DX < 0)
	case "zoom":
		return !c.Shift && (c.Key == mapKeyPlus && v.Scale == 2 || c.Key == mapKeyMinus && v.Scale == 0.5)
	default:
		return false
	}
}

func waitNativeState(ctx context.Context, driver nativeDriver, accept func(locationtrial.State) bool) (locationtrial.State, error) {
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return locationtrial.State{}, err
		}
		state, err := driver.State(ctx)
		if err != nil {
			return state, err
		}
		if accept(state) {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-tick.C:
		}
	}
}

func nativeInput(ctx context.Context, driver nativeDriver, command nativeCommand) (nativeObservation, error) {
	if err := ctx.Err(); err != nil {
		return nativeObservation{}, err
	}
	result, err := driver.Input(ctx, command)
	if err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, errors.New(result.Error)
	}
	if result.Kind != command.Kind || result.Skipped || !validNativeResponseTime(result.InputNS, result.VisibleNS) {
		return result, errors.New("native input has no observed visible response")
	}
	if (command.Kind == "cancel" || command.Kind == "close") && !result.ClosedViewer {
		return result, errors.New("native exit has no identified closed-viewer frame")
	}
	if (command.Kind == "pan" || command.Kind == "zoom") && (!result.Identified || !command.matchesTransform(result.Transform)) {
		return result, errors.New("native gesture has no identified transform; changed pixels alone cannot qualify latency")
	}
	return result, nil
}

// collectNative uses recorded UI stages for scan accounting only. Every gesture
// and cancellation latency comes from the independent screen/input driver.
func collectNative(ctx context.Context, driver nativeDriver, report *Report, sustain func() bool) error {
	if _, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool { return s.Ready }); err != nil {
		return err
	}
	open := func(sequence int) error {
		_, err := nativeInput(ctx, driver, nativeCommand{Kind: "open", Key: mapKeyL, Shift: true, Name: fmt.Sprintf("open-%02d", sequence)})
		return err
	}
	closeMap := func(sequence int) error {
		if _, err := nativeInput(ctx, driver, nativeCommand{Kind: "close", Key: mapKeyEscape, Name: fmt.Sprintf("close-%02d", sequence)}); err != nil {
			return err
		}
		if _, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool { return !s.Active && !s.Visible }); err != nil {
			return err
		}
		report.OpenCloseCycles++
		return nil
	}
	for sequence := range 2 {
		if err := open(sequence); err != nil {
			return err
		}
		state, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool {
			return s.Active && len(s.Stages) > sequence && s.Stages[sequence].Complete
		})
		if err != nil {
			return err
		}
		report.Images, report.Formats = state.Images, maps.Clone(state.Formats)
		report.Stages = append(report.Stages, state.Stages[sequence])
		if sequence == 0 {
			if err := closeMap(sequence); err != nil {
				return err
			}
		}
	}
	for i := 0; i < 40 || report.Images == 30_000 && sustain(); i++ {
		observation, err := nativeInput(ctx, driver, nativeGestureCommand(i))
		// An errored observation is retained, including a zero/incomplete sample.
		//goland:noinspection GoDfaErrorMayBeNotNil
		report.Gestures = append(report.Gestures, observation.Gesture)
		if err != nil {
			return fmt.Errorf("gesture %d: %w", i, err)
		}
	}
	if err := closeMap(1); err != nil {
		return err
	}
	sequence := 2
	if report.Images == 30_000 {
		if err := open(sequence); err != nil {
			return err
		}
		state, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool { return len(s.Stages) > sequence && s.Stages[sequence].Complete })
		if err != nil {
			return err
		}
		report.Stages = append(report.Stages, state.Stages[sequence])
		if err := closeMap(sequence); err != nil {
			return err
		}
		sequence++
	}
	if err := open(sequence); err != nil {
		return err
	}
	if _, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool { return s.Active && s.Visible }); err != nil {
		return err
	}
	observation, err := nativeInput(ctx, driver, nativeCommand{Kind: "cancel", Key: mapKeyEscape, Name: "cancellation"})
	// Preserve failed cancellation evidence instead of filtering it out.
	//goland:noinspection GoDfaErrorMayBeNotNil
	report.Cancellations = append(report.Cancellations, Cancellation{InputNS: observation.InputNS, VisibleNS: observation.VisibleNS, Before: observation.Before, After: observation.After, Complete: err == nil})
	if err != nil {
		return fmt.Errorf("cancellation: %w", err)
	}
	if _, err := waitNativeState(ctx, driver, func(s locationtrial.State) bool { return !s.Active && !s.Visible }); err != nil {
		return err
	}
	report.OpenCloseCycles++
	return nil
}
