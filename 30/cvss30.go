package gocvss30

import (
	"math"
	"strings"
	"unsafe"
)

// This file is based on https://www.first.org/cvss/v3.0/cvss-v30-specification_v1.9.pdf.

const (
	header = "CVSS:3.0/"
)

// ParseVector parses a given vector string, validates it
// and returns a CVSS31.
func ParseVector(vector string) (*CVSS30, error) {
	// Check header
	if !strings.HasPrefix(vector, header) {
		return nil, ErrInvalidCVSSHeader
	}
	vector = vector[len(header):]

	// Allocate CVSS v3.1 object
	cvss31 := &CVSS30{
		u0: 0,
		u1: 0,
		u2: 0,
		u3: 0,
		u4: 0,
		u5: 0, // last 4 bits are not used
	}

	// Parse vector
	kvm := kvm{}
	start := 0
	l := len(vector)
	for i := 0; i <= l; i++ {
		if i == l || vector[i] == '/' {
			a, v := splitCouple(vector[start:i])
			if err := kvm.Set(a); err != nil {
				return nil, err
			}
			if err := cvss31.Set(a, v); err != nil {
				return nil, err
			}
			start = i + 1
		}
	}

	// Check all base score metrics are defined
	if !kvm.av {
		return nil, &ErrMissing{Abv: "AV"}
	}
	if !kvm.ac {
		return nil, &ErrMissing{Abv: "AC"}
	}
	if !kvm.pr {
		return nil, &ErrMissing{Abv: "PR"}
	}
	if !kvm.ui {
		return nil, &ErrMissing{Abv: "UI"}
	}
	if !kvm.s {
		return nil, &ErrMissing{Abv: "S"}
	}
	if !kvm.c {
		return nil, &ErrMissing{Abv: "C"}
	}
	if !kvm.i {
		return nil, &ErrMissing{Abv: "I"}
	}
	if !kvm.a {
		return nil, &ErrMissing{Abv: "A"}
	}

	return cvss31, nil
}

// splitCouple is more efficient than `strings.Cut` as it is
// specialised on the ':' char.
func splitCouple(couple string) (string, string) {
	for i := 0; i < len(couple); i++ {
		if couple[i] == ':' {
			return couple[:i], couple[i+1:]
		}
	}
	return couple, ""
}

// Vector returns the CVSS v3.1 vector string representation.
func (cvss31 CVSS30) Vector() string {
	l := lenVec(&cvss31)
	b := make([]byte, 0, l)
	b = append(b, header...)

	// Base
	mandatory(&b, "AV:", cvss31.get("AV"))
	mandatory(&b, "/AC:", cvss31.get("AC"))
	mandatory(&b, "/PR:", cvss31.get("PR"))
	mandatory(&b, "/UI:", cvss31.get("UI"))
	mandatory(&b, "/S:", cvss31.get("S"))
	mandatory(&b, "/C:", cvss31.get("C"))
	mandatory(&b, "/I:", cvss31.get("I"))
	mandatory(&b, "/A:", cvss31.get("A"))

	// Temporal
	notMandatory(&b, "/E:", cvss31.get("E"))
	notMandatory(&b, "/RL:", cvss31.get("RL"))
	notMandatory(&b, "/RC:", cvss31.get("RC"))

	// Environmental
	notMandatory(&b, "/CR:", cvss31.get("CR"))
	notMandatory(&b, "/IR:", cvss31.get("IR"))
	notMandatory(&b, "/AR:", cvss31.get("AR"))
	notMandatory(&b, "/MAV:", cvss31.get("MAV"))
	notMandatory(&b, "/MAC:", cvss31.get("MAC"))
	notMandatory(&b, "/MPR:", cvss31.get("MPR"))
	notMandatory(&b, "/MUI:", cvss31.get("MUI"))
	notMandatory(&b, "/MS:", cvss31.get("MS"))
	notMandatory(&b, "/MC:", cvss31.get("MC"))
	notMandatory(&b, "/MI:", cvss31.get("MI"))
	notMandatory(&b, "/MA:", cvss31.get("MA"))

	return *(*string)(unsafe.Pointer(&b))
}

func lenVec(cvss31 *CVSS30) int {
	// Header: constant, so fixed (9)
	// Base:
	// - AV, AC, PR, UI: 4
	// - S, C, I, A: 3
	// - separators: 7
	// Total: 4*4 + 4*3 + 7 = 35
	l := len(header) + 35

	// Temporal:
	// - E: 3
	// - RL, RC: 4
	// - each one adds a separator
	if cvss31.get("E") != "X" {
		l += 4
	}
	if cvss31.get("RL") != "X" {
		l += 5
	}
	if cvss31.get("RC") != "X" {
		l += 5
	}

	// Environmental
	// - CR, IR, AR, MS, MC, MI, MA: 4
	// - MAV, MAC, MPR, MUI: 5
	// - each one adds a separator
	if cvss31.get("CR") != "X" {
		l += 5
	}
	if cvss31.get("IR") != "X" {
		l += 5
	}
	if cvss31.get("AR") != "X" {
		l += 5
	}
	if cvss31.get("MS") != "X" {
		l += 5
	}
	if cvss31.get("MC") != "X" {
		l += 5
	}
	if cvss31.get("MI") != "X" {
		l += 5
	}
	if cvss31.get("MA") != "X" {
		l += 5
	}
	if cvss31.get("MAV") != "X" {
		l += 6
	}
	if cvss31.get("MAC") != "X" {
		l += 6
	}
	if cvss31.get("MPR") != "X" {
		l += 6
	}
	if cvss31.get("MUI") != "X" {
		l += 6
	}

	return l
}

func mandatory(b *[]byte, pre, v string) {
	*b = append(*b, pre...)
	*b = append(*b, v...)
}

func notMandatory(b *[]byte, pre, v string) {
	if v == "X" {
		return
	}
	mandatory(b, pre, v)
}

// CVSS30 embeds all the metric values defined by the CVSS v3.1
// specification.
type CVSS30 struct {
	u0, u1, u2, u3, u4, u5 uint8
}

// Get returns the value of the given metric abbreviation.
func (cvss31 CVSS30) Get(abv string) (r string, err error) {
	switch abv {
	// Base
	case "AV":
		v := (cvss31.u0 & 0b11000000) >> 6
		switch v {
		case av_n:
			r = "N"
		case av_a:
			r = "A"
		case av_l:
			r = "L"
		case av_p:
			r = "P"
		}
	case "AC":
		v := (cvss31.u0 & 0b00100000) >> 5
		switch v {
		case ac_l:
			r = "L"
		case ac_h:
			r = "H"
		}
	case "PR":
		v := (cvss31.u0 & 0b00011000) >> 3
		switch v {
		case pr_n:
			r = "N"
		case pr_l:
			r = "L"
		case pr_h:
			r = "H"
		}
	case "UI":
		v := (cvss31.u0 & 0b00000100) >> 2
		switch v {
		case ui_n:
			r = "N"
		case ui_r:
			r = "R"
		}
	case "S":
		v := (cvss31.u0 & 0b00000010) >> 1
		switch v {
		case s_u:
			r = "U"
		case s_c:
			r = "C"
		}
	case "C":
		v := ((cvss31.u0 & 0b00000001) << 1) | (cvss31.u1&0b10000000)>>7
		switch v {
		case cia_h:
			r = "H"
		case cia_l:
			r = "L"
		case cia_n:
			r = "N"
		}
	case "I":
		v := (cvss31.u1 & 0b01100000) >> 5
		switch v {
		case cia_h:
			r = "H"
		case cia_l:
			r = "L"
		case cia_n:
			r = "N"
		}
	case "A":
		v := (cvss31.u1 & 0b00011000) >> 3
		switch v {
		case cia_h:
			r = "H"
		case cia_l:
			r = "L"
		case cia_n:
			r = "N"
		}

	// Temporal
	case "E":
		v := cvss31.u1 & 0b00000111
		switch v {
		case e_x:
			r = "X"
		case e_h:
			r = "H"
		case e_f:
			r = "F"
		case e_p:
			r = "P"
		case e_u:
			r = "U"
		}
	case "RL":
		v := (cvss31.u2 & 0b11100000) >> 5
		switch v {
		case rl_x:
			r = "X"
		case rl_u:
			r = "U"
		case rl_w:
			r = "W"
		case rl_t:
			r = "T"
		case rl_o:
			r = "O"
		}
	case "RC":
		v := (cvss31.u2 & 0b00011000) >> 3
		switch v {
		case rc_x:
			r = "X"
		case rc_c:
			r = "C"
		case rc_r:
			r = "R"
		case rc_u:
			r = "U"
		}

	// Environmental
	case "CR":
		v := (cvss31.u2 & 0b00000110) >> 1
		switch v {
		case ciar_x:
			r = "X"
		case ciar_h:
			r = "H"
		case ciar_m:
			r = "M"
		case ciar_l:
			r = "L"
		}
	case "IR":
		v := ((cvss31.u2 & 0b00000001) << 1) | ((cvss31.u3 & 0b10000000) >> 7)
		switch v {
		case ciar_x:
			r = "X"
		case ciar_h:
			r = "H"
		case ciar_m:
			r = "M"
		case ciar_l:
			r = "L"
		}
	case "AR":
		v := (cvss31.u3 & 0b01100000) >> 5
		switch v {
		case ciar_x:
			r = "X"
		case ciar_h:
			r = "H"
		case ciar_m:
			r = "M"
		case ciar_l:
			r = "L"
		}
	case "MAV":
		v := (cvss31.u3 & 0b00011100) >> 2
		switch v {
		case mav_x:
			r = "X"
		case mav_n:
			r = "N"
		case mav_a:
			r = "A"
		case mav_l:
			r = "L"
		case mav_p:
			r = "P"
		}
	case "MAC":
		v := cvss31.u3 & 0b00000011
		switch v {
		case mac_x:
			r = "X"
		case mac_l:
			r = "L"
		case mac_h:
			r = "H"
		}
	case "MPR":
		v := (cvss31.u4 & 0b11000000) >> 6
		switch v {
		case mpr_x:
			r = "X"
		case mpr_n:
			r = "N"
		case mpr_l:
			r = "L"
		case mpr_h:
			r = "H"
		}
	case "MUI":
		v := (cvss31.u4 & 0b00110000) >> 4
		switch v {
		case mui_x:
			r = "X"
		case mui_n:
			r = "N"
		case mui_r:
			r = "R"
		}
	case "MS":
		v := (cvss31.u4 & 0b00001100) >> 2
		switch v {
		case ms_x:
			r = "X"
		case ms_u:
			r = "U"
		case ms_c:
			r = "C"
		}
	case "MC":
		v := cvss31.u4 & 0b00000011
		switch v {
		case mcia_x:
			r = "X"
		case mcia_n:
			r = "N"
		case mcia_l:
			r = "L"
		case mcia_h:
			r = "H"
		}
	case "MI":
		v := (cvss31.u5 & 0b11000000) >> 6
		switch v {
		case mcia_x:
			r = "X"
		case mcia_n:
			r = "N"
		case mcia_l:
			r = "L"
		case mcia_h:
			r = "H"
		}
	case "MA":
		v := (cvss31.u5 & 0b00110000) >> 4
		switch v {
		case mcia_x:
			r = "X"
		case mcia_n:
			r = "N"
		case mcia_l:
			r = "L"
		case mcia_h:
			r = "H"
		}
	default:
		err = &ErrInvalidMetric{Abv: abv}
	}
	return
}

// Set sets the value of the given metric abbreviation.
func (cvss31 *CVSS30) Set(abv string, value string) error {
	switch abv {
	// Base
	case "AV":
		v, err := validate(value, []string{"N", "A", "L", "P"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b00111111) | (v << 6)
	case "AC":
		v, err := validate(value, []string{"L", "H"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b11011111) | (v << 5)
	case "PR":
		v, err := validate(value, []string{"N", "L", "H"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b11100111) | (v << 3)
	case "UI":
		v, err := validate(value, []string{"N", "R"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b11111011) | (v << 2)
	case "S":
		v, err := validate(value, []string{"U", "C"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b11111101) | (v << 1)
	case "C":
		v, err := validate(value, []string{"H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u0 = (cvss31.u0 & 0b11111110) | ((v & 0b10) >> 1)
		cvss31.u1 = (cvss31.u1 & 0b01111111) | ((v & 0b01) << 7)
	case "I":
		v, err := validate(value, []string{"H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u1 = (cvss31.u1 & 0b10011111) | (v << 5)
	case "A":
		v, err := validate(value, []string{"H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u1 = (cvss31.u1 & 0b11100111) | (v << 3)

	// Temporal
	case "E":
		v, err := validate(value, []string{"X", "H", "F", "P", "U"})
		if err != nil {
			return err
		}
		cvss31.u1 = (cvss31.u1 & 0b11111000) | v
	case "RL":
		v, err := validate(value, []string{"X", "U", "W", "T", "O"})
		if err != nil {
			return err
		}
		cvss31.u2 = (cvss31.u2 & 0b00011111) | (v << 5)
	case "RC":
		v, err := validate(value, []string{"X", "C", "R", "U"})
		if err != nil {
			return err
		}
		cvss31.u2 = (cvss31.u2 & 0b11100111) | (v << 3)

	// Environmental
	case "CR":
		v, err := validate(value, []string{"X", "H", "M", "L"})
		if err != nil {
			return err
		}
		cvss31.u2 = (cvss31.u2 & 0b11111001) | (v << 1)
	case "IR":
		v, err := validate(value, []string{"X", "H", "M", "L"})
		if err != nil {
			return err
		}
		cvss31.u2 = (cvss31.u2 & 0b11111110) | ((v & 0b10) >> 1)
		cvss31.u3 = (cvss31.u3 & 0b01111111) | ((v & 0b01) << 7)
	case "AR":
		v, err := validate(value, []string{"X", "H", "M", "L"})
		if err != nil {
			return err
		}
		cvss31.u3 = (cvss31.u3 & 0b10011111) | (v << 5)
	case "MAV":
		v, err := validate(value, []string{"X", "N", "A", "L", "P"})
		if err != nil {
			return err
		}
		cvss31.u3 = (cvss31.u3 & 0b11100011) | (v << 2)
	case "MAC":
		v, err := validate(value, []string{"X", "L", "H"})
		if err != nil {
			return err
		}
		cvss31.u3 = (cvss31.u3 & 0b11111100) | v
	case "MPR":
		v, err := validate(value, []string{"X", "N", "L", "H"})
		if err != nil {
			return err
		}
		cvss31.u4 = (cvss31.u4 & 0b00111111) | (v << 6)
	case "MUI":
		v, err := validate(value, []string{"X", "N", "R"})
		if err != nil {
			return err
		}
		cvss31.u4 = (cvss31.u4 & 0b11001111) | (v << 4)
	case "MS":
		v, err := validate(value, []string{"X", "U", "C"})
		if err != nil {
			return err
		}
		cvss31.u4 = (cvss31.u4 & 0b11110011) | (v << 2)
	case "MC":
		v, err := validate(value, []string{"X", "H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u4 = (cvss31.u4 & 0b11111100) | v
	case "MI":
		v, err := validate(value, []string{"X", "H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u5 = (cvss31.u5 & 0b00111111) | (v << 6)
	case "MA":
		v, err := validate(value, []string{"X", "H", "L", "N"})
		if err != nil {
			return err
		}
		cvss31.u5 = (cvss31.u5 & 0b11000000) | (v << 4)
	default:
		return &ErrInvalidMetric{Abv: abv}
	}
	return nil
}

// validate returns the index of value in enabled if matches.
// enabled values have to match the values.go constants order.
func validate(value string, enabled []string) (i uint8, err error) {
	// Check is valid
	for _, enbl := range enabled {
		if value == enbl {
			return i, nil
		}
		i++
	}
	return 0, ErrInvalidMetricValue
}

// get is used for internal purposes only.
func (cvss31 CVSS30) get(abv string) string {
	str, _ := cvss31.Get(abv)
	return str
}

// BaseScore returns the CVSS v3.1's base score.
func (cvss31 CVSS30) BaseScore() float64 {
	impact := cvss31.Impact()
	exploitability := cvss31.Exploitability()
	if impact <= 0 {
		return 0
	}
	if v, _ := cvss31.Get("S"); v == "U" {
		return roundup(math.Min(impact+exploitability, 10))
	}
	return roundup(math.Min(1.08*(impact+exploitability), 10))
}

func (cvss31 CVSS30) Impact() float64 {
	iss := 1 - ((1 - cia(cvss31.get("C"))) * (1 - cia(cvss31.get("I"))) * (1 - cia(cvss31.get("A"))))
	if v, _ := cvss31.Get("S"); v == "U" {
		return 6.42 * iss
	}
	return 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
}

func (cvss31 CVSS30) Exploitability() float64 {
	return 8.22 * attackVector(cvss31.get("AV")) * attackComplexity(cvss31.get("AC")) * privilegesRequired(cvss31.get("PR"), cvss31.get("S")) * userInteraction(cvss31.get("UI"))
}

// TemporalScore returns the CVSS v3.1's temporal score.
func (cvss31 CVSS30) TemporalScore() float64 {
	return roundup(cvss31.BaseScore() * exploitCodeMaturity(cvss31.get("E")) * remediationLevel(cvss31.get("RL")) * reportConfidence(cvss31.get("RC")))
}

// EnvironmentalScore returns the CVSS v3.1's environmental score.
func (cvss31 CVSS30) EnvironmentalScore() float64 {
	// Choose which to use (use base if modified is not defined).
	// It is based on first.org online calculator's source code,
	// while it is not explicit in the specification which value
	// to use.
	mav := mod(cvss31.get("AV"), cvss31.get("MAV"))
	mac := mod(cvss31.get("AC"), cvss31.get("MAC"))
	mpr := mod(cvss31.get("PR"), cvss31.get("MPR"))
	mui := mod(cvss31.get("UI"), cvss31.get("MUI"))
	ms := mod(cvss31.get("S"), cvss31.get("MS"))
	mc := mod(cvss31.get("C"), cvss31.get("MC"))
	mi := mod(cvss31.get("I"), cvss31.get("MI"))
	ma := mod(cvss31.get("A"), cvss31.get("MA"))

	miss := math.Min(1-(1-ciar(cvss31.get("CR"))*cia(mc))*(1-ciar(cvss31.get("IR"))*cia(mi))*(1-ciar(cvss31.get("AR"))*cia(ma)), 0.915)
	var modifiedImpact float64
	if ms == "U" {
		modifiedImpact = 6.42 * miss
	} else {
		modifiedImpact = 7.52*(miss-0.029) - 3.25*math.Pow(miss-0.02, 15)
	}
	modifiedExploitability := 8.22 * attackVector(mav) * attackComplexity(mac) * privilegesRequired(mpr, ms) * userInteraction(mui)
	if modifiedImpact <= 0 {
		return 0
	}
	if ms == "U" {
		return roundup(roundup(math.Min(modifiedImpact+modifiedExploitability, 10)) * exploitCodeMaturity(cvss31.get("E")) * remediationLevel(cvss31.get("RL")) * reportConfidence(cvss31.get("RC")))
	}
	r := math.Min(1.08*(modifiedImpact+modifiedExploitability), 10)
	return roundup(roundup(r) * exploitCodeMaturity(cvss31.get("E")) * remediationLevel(cvss31.get("RL")) * reportConfidence(cvss31.get("RC")))
}

// Rating returns the verbose for a given rating.
// It does not check wether the number of decimal is valid,
// as it can differ due to binary imprecisions, and such
// behaviour is not enforced by the specification.
func Rating(score float64) (string, error) {
	if score < 0.0 || score > 10.0 {
		return "", ErrOutOfBoundsScore
	}
	if score >= 9.0 {
		return "CRITICAL", nil
	}
	if score >= 7.0 {
		return "HIGH", nil
	}
	if score >= 4.0 {
		return "MEDIUM", nil
	}
	if score >= 0.1 {
		return "LOW", nil
	}
	return "NONE", nil
}

// Helpers to compute CVSS v3.1 scores

func attackVector(v string) float64 {
	switch v {
	case "N":
		return 0.85
	case "A":
		return 0.62
	case "L":
		return 0.55
	case "P":
		return 0.2
	default:
		panic(ErrInvalidMetricValue)
	}
}

func attackComplexity(v string) float64 {
	switch v {
	case "L":
		return 0.77
	case "H":
		return 0.44
	default:
		panic(ErrInvalidMetricValue)
	}
}

func privilegesRequired(v, scope string) float64 {
	switch v {
	case "N":
		return 0.85
	case "L":
		if scope == "C" {
			return 0.68
		}
		return 0.62
	case "H":
		if scope == "C" {
			return 0.5
		}
		return 0.27
	default:
		panic(ErrInvalidMetricValue)
	}
}

func userInteraction(v string) float64 {
	switch v {
	case "N":
		return 0.85
	case "R":
		return 0.62
	default:
		panic(ErrInvalidMetricValue)
	}
}

func cia(v string) float64 {
	switch v {
	case "H":
		return 0.56
	case "L":
		return 0.22
	case "N":
		return 0
	default:
		panic(ErrInvalidMetricValue)
	}
}

func exploitCodeMaturity(v string) float64 {
	switch v {
	case "X":
		return 1
	case "H":
		return 1
	case "F":
		return 0.97
	case "P":
		return 0.94
	case "U":
		return 0.91
	default:
		panic(ErrInvalidMetricValue)
	}
}

func remediationLevel(v string) float64 {
	switch v {
	case "X":
		return 1
	case "U":
		return 1
	case "W":
		return 0.97
	case "T":
		return 0.96
	case "O":
		return 0.95
	default:
		panic(ErrInvalidMetricValue)
	}
}

func reportConfidence(v string) float64 {
	switch v {
	case "X":
		return 1
	case "C":
		return 1
	case "R":
		return 0.96
	case "U":
		return 0.92
	default:
		panic(ErrInvalidMetricValue)
	}
}

func ciar(v string) float64 {
	switch v {
	case "X":
		return 1
	case "H":
		return 1.5
	case "M":
		return 1
	case "L":
		return 0.5
	default:
		panic(ErrInvalidMetricValue)
	}
}

func roundup(x float64) float64 {
	bx := math.RoundToEven(x * 100000)
	if int(bx)%10000 == 0 {
		return bx / 100000.0
	}
	return (math.Floor(bx/10000) + 1) / 10.0
}

func mod(base, modified string) string {
	if modified != "X" {
		return modified
	}
	return base
}

// kvm stands for Key-Value Map, and is used to make sure each
// metric is defined only once, as documented by the CVSS v3.1
// specification document, section 6 "Vector String" paragraph 3.
// Using this avoids a map that escapes to heap for each call of
// ParseVector, as its size is known and wont evolve.
type kvm struct {
	// base metrics
	av, ac, pr, ui, s, c, i, a bool
	// temporal metrics
	e, rl, rc bool
	// environmental metrics
	cr, ir, ar, mav, mac, mpr, mui, ms, mc, mi, ma bool
}

func (kvm *kvm) Set(abv string) error {
	var dst *bool
	switch abv {
	case "AV":
		dst = &kvm.av
	case "AC":
		dst = &kvm.ac
	case "PR":
		dst = &kvm.pr
	case "UI":
		dst = &kvm.ui
	case "S":
		dst = &kvm.s
	case "C":
		dst = &kvm.c
	case "I":
		dst = &kvm.i
	case "A":
		dst = &kvm.a
	case "E":
		dst = &kvm.e
	case "RL":
		dst = &kvm.rl
	case "RC":
		dst = &kvm.rc
	case "CR":
		dst = &kvm.cr
	case "IR":
		dst = &kvm.ir
	case "AR":
		dst = &kvm.ar
	case "MAV":
		dst = &kvm.mav
	case "MAC":
		dst = &kvm.mac
	case "MPR":
		dst = &kvm.mpr
	case "MUI":
		dst = &kvm.mui
	case "MS":
		dst = &kvm.ms
	case "MC":
		dst = &kvm.mc
	case "MI":
		dst = &kvm.mi
	case "MA":
		dst = &kvm.ma
	default:
		return &ErrInvalidMetric{Abv: abv}
	}
	if *dst {
		return &ErrDefinedN{Abv: abv}
	}
	*dst = true
	return nil
}
