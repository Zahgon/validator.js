package validator_test

import "testing"

// TestValidators ports test/validators.test.js.
func TestValidators(t *testing.T) { runFixture(t, "validators") }

// TestSanitizers ports test/sanitizers.test.js.
func TestSanitizers(t *testing.T) { runFixture(t, "sanitizers") }

// The per-validator suites of test/validators/.

func TestIsAfter(t *testing.T)          { runFixture(t, "isAfter") }
func TestIsBase64(t *testing.T)         { runFixture(t, "isBase64") }
func TestIsBefore(t *testing.T)         { runFixture(t, "isBefore") }
func TestIsFQDN(t *testing.T)           { runFixture(t, "isFQDN") }
func TestIsIP(t *testing.T)             { runFixture(t, "isIP") }
func TestIsISBN(t *testing.T)           { runFixture(t, "isISBN") }
func TestIsISO31661Alpha2(t *testing.T) { runFixture(t, "isISO31661Alpha2") }
func TestIsISO31661Alpha3(t *testing.T) { runFixture(t, "isISO31661Alpha3") }
func TestIsLength(t *testing.T)         { runFixture(t, "isLength") }
