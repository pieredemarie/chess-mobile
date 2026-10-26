package service

import (
	"errors"
	"time"
)

var ErrTimeExpired = errors.New("time expired")

type Clock struct {
	White, Black time.Duration
	Increment    time.Duration
	Turn         string
	Since        time.Time
}

func NewClock(base, increment time.Duration, now time.Time) (Clock, error) {
	if base <= 0 || increment < 0 {
		return Clock{}, errors.New("invalid time control")
	}
	return Clock{White: base, Black: base, Increment: increment, Turn: "w", Since: now}, nil
}

func (c Clock) Remaining(now time.Time) (time.Duration, time.Duration) {
	w, b := c.White, c.Black
	elapsed := now.Sub(c.Since)
	if elapsed < 0 {
		elapsed = 0
	}
	if c.Turn == "w" {
		w -= elapsed
	} else {
		b -= elapsed
	}
	if w < 0 {
		w = 0
	}
	if b < 0 {
		b = 0
	}
	return w, b
}

func (c *Clock) Move(now time.Time) error {
	w, b := c.Remaining(now)
	if (c.Turn == "w" && w == 0) || (c.Turn == "b" && b == 0) {
		return ErrTimeExpired
	}
	if c.Turn == "w" {
		c.White, c.Black, c.Turn = w+c.Increment, b, "b"
	} else {
		c.White, c.Black, c.Turn = w, b+c.Increment, "w"
	}
	c.Since = now
	return nil
}

func (c Clock) Deadline() time.Time {
	if c.Turn == "w" {
		return c.Since.Add(c.White)
	}
	return c.Since.Add(c.Black)
}
