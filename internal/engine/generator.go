package engine

import (
	"math"
	"math/rand/v2"

	"github.com/ausanyal/firesage/internal/data"
)

// generatePathScenario produces a sequence of numYears MarketYear observations for a single simulation path
// according to the selected SimulationMode.
func generatePathScenario(
	mode SimulationMode,
	country data.CountryCode,
	history []data.MarketYear,
	numYears int,
	pathIdx int,
	rng *rand.Rand,
) []data.MarketYear {
	nHist := len(history)
	out := make([]data.MarketYear, numYears)
	if nHist == 0 || numYears == 0 {
		return out
	}

	switch mode {
	case ModeHistorical:
		// Deterministic rolling cohort starting at pathIdx % nHist (wrapping circularly for 50+ yr horizons)
		startIdx := pathIdx % nHist
		for y := 0; y < numYears; y++ {
			out[y] = history[(startIdx+y)%nHist]
		}
		return out

	case ModeRegimeMC:
		// 3-State Markov Regime-Switching Model (0=Expansion, 1=Deflationary Crash, 2=Stagflation)
		// with Valuation/CAPE mean-reversion conditioning over the first 12 years.
		currPE, medianPE := data.CurrentValuation(country)
		valuationDrag := 0.0
		if currPE > 0 && medianPE > 0 {
			// Karsten Jeske / Shiller CAPE earnings-yield tilt: overvalued markets experience muted 10y real drift
			valuationDrag = (1.0/currPE - 1.0/medianPE) * 0.65
		}

		// Transition matrix rows: [toExpansion, toCrash, toStagflation]
		transitions := [3][3]float64{
			{0.80, 0.10, 0.10}, // From Expansion
			{0.62, 0.23, 0.15}, // From Crash
			{0.35, 0.12, 0.53}, // From Stagflation (sticky!)
		}

		state := 0
		u0 := rng.Float64()
		if currPE > medianPE*1.35 {
			// Elevated starting CAPE slightly increases early regime transition risk
			if u0 < 0.16 {
				state = 1
			} else if u0 < 0.30 {
				state = 2
			}
		}

		isUS := country == data.CountryUSA
		simPE := currPE
		for y := 0; y < numYears; y++ {
			var eqMean, eqVol, debtMean, debtVol, cashRate, goldMean, goldVol, cpiMean, cpiVol, medSpread float64
			if isUS {
				switch state {
				case 0: // Expansion
					eqMean, eqVol = 0.128, 0.135
					debtMean, debtVol = 0.048, 0.065
					cashRate = 0.035
					goldMean, goldVol = 0.045, 0.13
					cpiMean, cpiVol = 0.026, 0.011
					medSpread = 0.016
				case 1: // Crash / Recession
					eqMean, eqVol = -0.215, 0.220
					debtMean, debtVol = 0.095, 0.085 // Flight to Treasuries
					cashRate = 0.018
					goldMean, goldVol = 0.095, 0.16
					cpiMean, cpiVol = 0.008, 0.018
					medSpread = 0.018
				default: // Stagflation
					eqMean, eqVol = 0.015, 0.175
					debtMean, debtVol = -0.025, 0.095 // Rising yields hurt bonds
					cashRate = 0.068
					goldMean, goldVol = 0.185, 0.21
					cpiMean, cpiVol = 0.078, 0.024
					medSpread = 0.024
				}
			} else {
				// India regime parameters (higher nominal growth & structural CPI/medical inflation)
				switch state {
				case 0: // Expansion
					eqMean, eqVol = 0.165, 0.185
					debtMean, debtVol = 0.082, 0.048
					cashRate = 0.064
					goldMean, goldVol = 0.098, 0.13
					cpiMean, cpiVol = 0.052, 0.014
					medSpread = 0.068
				case 1: // Crash
					eqMean, eqVol = -0.245, 0.265
					debtMean, debtVol = 0.115, 0.062
					cashRate = 0.055
					goldMean, goldVol = 0.175, 0.16
					cpiMean, cpiVol = 0.058, 0.018
					medSpread = 0.070
				default: // Indian Stagflation (1991-92 / 2010-13 style)
					eqMean, eqVol = 0.042, 0.210
					debtMean, debtVol = 0.045, 0.058
					cashRate = 0.082
					goldMean, goldVol = 0.215, 0.18
					cpiMean, cpiVol = 0.104, 0.022
					medSpread = 0.048
				}
			}

			// Apply decaying valuation conditioning over first 12 years
			valAdj := 0.0
			if y < 12 {
				valAdj = valuationDrag * (1.0 - float64(y)/12.0)
			}

			zEq := rng.NormFloat64()
			zDebt := -0.22*zEq + math.Sqrt(1-0.22*0.22)*rng.NormFloat64()
			zGold := -0.15*zEq + math.Sqrt(1-0.15*0.15)*rng.NormFloat64()
			zCPI := rng.NormFloat64()

			// Occasional Merton fat-tail jump (4% annual crash jump risk)
			jump := 0.0
			if rng.Float64() < 0.04 {
				jump = -0.14 - math.Abs(rng.NormFloat64())*0.08
			}

			eqRet := clamp(eqMean+valAdj+eqVol*zEq+jump, -0.62, 0.95)
			debtRet := clamp(debtMean+debtVol*zDebt, -0.25, 0.40)
			cpi := clamp(cpiMean+cpiVol*zCPI, -0.04, 0.20)
			medCPI := clamp(cpi+medSpread+0.012*rng.NormFloat64(), 0.01, 0.24)
			goldRet := clamp(goldMean+goldVol*zGold+0.35*math.Max(0, cpi-cpiMean), -0.35, 0.90)

			// Evolve synthetic CAPE/PE for dynamic CAPE withdrawal strategy
			realEq := (1 + eqRet) / (1 + math.Max(-0.05, cpi))
			simPE = clamp(simPE*realEq*0.935+medianPE*0.065, 6.0, 50.0)

			out[y] = data.MarketYear{
				Year:        2026 + y,
				Equity:      eqRet,
				Debt:        debtRet,
				Cash:        cashRate,
				Gold:        goldRet,
				HeadlineCPI: cpi,
				MedicalCPI:  medCPI,
				ValuationPE: simPE,
			}

			// Sample next Markov regime
			u := rng.Float64()
			if u < transitions[state][0] {
				state = 0
			} else if u < transitions[state][0]+transitions[state][1] {
				state = 1
			} else {
				state = 2
			}
		}
		return out

	default:
		// Politis & Romano (1994) Stationary Block Bootstrap:
		// Samples contiguous historical blocks with geometric length (mean block = 4.0 years, p = 0.25)
		// to preserve both contemporary cross-asset correlation AND multi-year stagflation/crash serial correlation.
		const restartProb = 0.25
		idx := rng.IntN(nHist)
		for y := 0; y < numYears; y++ {
			if y > 0 {
				if rng.Float64() < restartProb {
					idx = rng.IntN(nHist)
				} else {
					idx = (idx + 1) % nHist
				}
			}
			out[y] = history[idx]
		}
		return out
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
