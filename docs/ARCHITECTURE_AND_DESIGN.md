# FireSage — Product, Quantitative & Go Systems Architecture Blueprint

> **Mission:** Empower individuals in **India (`₹`)** and the **USA (`$`)** to answer *"When can I realistically retire?"* and *"Which life, tax, and city choices get me there safely?"* using institutional stochastic modeling, real regional cost & state-tax intelligence (e.g., **Bengaluru vs. Goa**, **San Jose, CA vs. Raleigh, NC**), and 60fps interactive visual storytelling.

---

## 1. Core Product Pillars

| Pillar | What Makes FireSage Best-in-Class |
| :--- | :--- |
| **Dual-Country Native Engine** | Purpose-built for **India (`₹ L / Cr`)** and **USA (`$K / $M`)** — never a superficial currency-symbol swap. Models exact tax laws, retirement account lock-ins, and healthcare structures for each country. |
| **City-Tier & Domestic Geo-Arbitrage** | Built-in calibrated city profiles and **"Earn in Tier-1 → Retire in Tier-2" transitions** (e.g., *Accumulate in Bengaluru → Retire in Goa*, or *Accumulate in San Jose, CA → Retire in Raleigh, NC*), capturing housing shifts, US state tax drops, and regional healthcare curves. |
| **4-Engine Stochastic Rigor** | Goroutine-parallel Stationary Block Bootstrap (`10,000` paths in `~10ms` preserving fat tails & multi-year stagflation), Historical Crisis Replay (1871+ US / 1979+ India), Valuation/CAPE-Conditioned Regime-Switching MC, and Black-Swan Stress Tests. |
| **Liquidity vs. Net Worth Reality** | Separates *total net worth* from *spendable early-retirement liquidity* — catching pre-59½ / pre-60 depletion traps when funds are locked in 401(k)s, EPF, or NPS. |
| **60-Second Onboarding → Infinite Depth** | 3-tier progressive disclosure: 5 inputs yield an instant animated probability cone in `<60s`; expandable cards unlock multi-bucket tax sequencing, guardrails, and life milestones. |
| **Local-First + Cloud-Extensible** | Zero-signup private `IndexedDB` persistence + JSON export/import in V1, built behind a clean `PlanRepository` interface ready for **Google Sign-In + Cloud Firestore** sync. |

---

## 2. City-Tier & Regional Geo-Arbitrage Engine (India & USA)

Where you retire changes *three* variables simultaneously: **baseline living basket**, **taxes**, and **healthcare/housing mechanics**. FireSage treats location as a first-class simulation vector:

### 2.1 India City-Tier Profiles (`₹`)
* **Tier-1 Metros:** *Bengaluru, Mumbai, Delhi-NCR, Hyderabad, Pune, Chennai*
  * Higher housing rent / society maintenance, domestic help & urban services basket, Metro Zone-1 health insurance premiums, higher lifestyle creep pressure.
* **Tier-2 & Lifestyle Retirement Cities:** *Goa, Mysuru, Coimbatore, Kochi, Chandigarh, Jaipur, Dehradun, Pondicherry*
  * **30%–55% lower housing/rental cost** vs Tier-1 metros; distinct discretionary & travel basket; Zone-2/3 retail health insurance pricing paired with an optional *"Metro Tertiary Medical Travel Buffer"*.
* **Domestic Geo-Arbitrage Transition:**
  * Model *"Work in Bengaluru until Age 44 → Relocate & Retire in Goa"* (including optional primary home equity release / rental yield conversion or buying a retirement home at FIRE date).

### 2.2 USA City & State Profiles (`$`)
* **Tier-1 / VHCOL Metros:** *San Jose / SF Bay Area (CA), New York City (NY), Seattle (WA), Boston (MA)*
  * Models VHCOL housing/property tax/living baskets **plus exact state income & capital gains taxes** (e.g., California taxing long-term capital gains and Roth conversions as ordinary income up to `13.3%+`).
* **Tier-2 / MCOL & Retirement Hubs:** *Raleigh / Research Triangle (NC), Austin (TX), Tampa (FL), Denver (CO), Phoenix (AZ), Nashville (TN)*
  * Captures **MCOL expense scaling**, **State Tax Arbitrage** (e.g., moving from San Jose, CA `13.3%` top bracket to Raleigh, NC `~4.25%` flat rate or TX/FL `0%` state income tax), property tax rate differences, and **region-adjusted ACA Marketplace Silver benchmark premiums**.
* **Domestic Geo-Arbitrage Transition:**
  * Model *"Accumulate at Bay Area comp in San Jose, CA → Retire in Raleigh, NC at Age 45"*, quantifying exactly how many years earlier retirement becomes viable from combined **cost-of-living + CA→NC state tax savings + home equity relocation**.

---

## 3. Gold-Standard Data Sources & Tax/Regulatory Rules

### 3.1 United States (`$`)
* **Historical Market & Macro Data:** **Robert Shiller Yale Dataset** (1871–present monthly S&P Composite, dividends, **CAPE**, 10Y Treasury, CPI-U) + **Aswath Damodaran NYU Stern Dataset** (1928–present S&P 500, Small Cap, T-Bills, 10Y Treasuries, Corporate Bonds, Gold, Real Estate) + **FRED** macro series.
* **Tax, Early Access & Healthcare (2026 Rules):**
  * Federal + State ordinary & LTCG tax brackets (`0% / 15% / 20%` + `3.8%` NIIT).
  * **Pre-59½ Access:** Taxable bridge (basis tracking), **5-Year Roth Conversion Ladder**, **Rule of 55**, and **72(t) SEPP**.
  * **2026 ACA Healthcare Cliff:** Models the restored **hard 400% FPL ACA Subsidy Cliff**, dynamically coupling MAGI (from Roth conversions + capital gains) to pre-65 health insurance costs before switching to **Medicare + IRMAA** at age 65.
  * **SECURE 2.0 & Social Security:** Age 73/75 RMDs, Roth 401(k) RMD exemption, SSA **PIA Bend Points** (claiming ages 62–70).

### 3.2 India (`₹`)
* **Historical Market & Macro Data:** **Nifty 50 TRI** (1995–present), **Nifty Next 50 / Midcap 150 TRI**, spliced **S&P BSE Sensex + dividend yield** back to 1979; **RBI DBIE & CCIL** 10Y G-Sec & T-Bill series; **INR Gold**; **MoSPI CPI-Combined** separated from **Medical Inflation (12%–14% p.a.)** and **Lifestyle Inflation (8%–9% p.a.)**.
* **Tax, Lock-in & Healthcare (2026 Rules):**
  * **Capital Gains:** Equity LTCG at **12.5%** above ₹1.25L annual exemption; STCG **20%**; Debt MFs/FDs at **slab rates**; **Arbitrage Funds** with equity tax treatment for liquid/short-term buckets.
  * **Retirement Lock-ins:** **EPF/VPF**, **PPF** (EEE, 15-year + 5-year blocks), **NPS Tier-I** (age 60 lock-in, 60% tax-free lump sum / SLW + 40% taxable annuity, Employer `80CCD(2)`).
  * **Self-Funded Healthcare:** Escalating retail family floater + Super Top-up age-band premiums + medical corpus buffer.

---

## 4. Quantitative Simulation Engine (Concurrent Go Core)

1. **Stationary Block Bootstrap (Default):** Worker-pool goroutines sample geometric-length contiguous multi-year blocks of joint asset returns & inflation `(equity, smid, debt, cash, gold, headline_cpi, medical_cpi)` with zero-allocation slice reuse (`sync.Pool`), running `10,000` 60-year trajectories in `<15ms`.
2. **CAPE / Valuation-Conditioned Regime-Switching MC:** 3-regime Markov model (*Expansion, Crash, Stagflation*) conditioned on starting Shiller CAPE (US) or Nifty P/E percentile (India).
3. **Historical Cohort Backtester:** Replays every historical retirement start year (1871+ US; 1979+ India), pinpointing worst cohorts (1929, 1966, 1973, 2000, 2008; 1992, 2000, 2008, 2010).
4. **Custom Black-Swan Stress Lab:** Explicit user-injected market, inflation, longevity, or medical shocks.

---

## 5. Signature Visualizations (React 19 + TypeScript + ECharts/Canvas)

1. **Stochastic Wealth Cone & Trajectory Explorer:** `P5–P90` bands + ghosted paths + **draggable FIRE Age playhead** + click-any-path year-by-year crisis autopsy.
2. **"When Can I Retire?" Readiness Frontier:** Plots Retirement Age vs. Survival Probability & Safe Spend, pinning **Coast FIRE**, **Lean FIRE**, **Target FIRE (90%)**, and **Fortress FIRE (98%)** ages.
3. **City Geo-Arbitrage Impact Split-View:** Instant delta comparing *"Stay & Retire in San Jose / Bengaluru"* vs *"Relocate & Retire in Raleigh / Goa"* (years saved, corpus saved, state/local tax saved).
4. **Lifespan Cash-Flow & Tax Streamgraph / Sankey:** Tracks cash flows across Salary → Taxable/Arbitrage Bridge → Roth Ladder → 401k/EPF/NPS → Social Security, highlighting any pre-60 **Liquidity Gap**.
5. **Sequence-of-Returns "Danger Zone" Heatmap & Bond-Tent Shield.**
6. **"Living Reality" Dynamic Guardrail Spending Bands.**
7. **Historical Crisis Stress-Test Leaderboard.**
8. **Goal-Seek Trade-off Contour Map & Actionable Levers Tornado Chart.**

---

## 6. Locked-In Technical Stack & Repository Structure

* **Backend & Quant Engine:** **Go (`net/http` stdlib router + worker-pool goroutine Monte Carlo engine)**
* **Frontend:** **React 19 + TypeScript + Vite + Tailwind CSS + Apache ECharts / Custom Canvas**
* **Single Static Binary Packaging:** Compiled Vite `dist/` embedded directly into the Go binary via `//go:embed`, deployed via a root `Dockerfile` (`PORT=8080`, `gcr.io/distroless/static-debian12`) to **Google Cloud Run** (mirroring `StockSage`'s GitHub → Cloud Run deployment flow).
* **Persistence:** Local-First `IndexedDB` (`PlanRepository` interface) + versioned JSON import/export + URL snapshot sharing, cleanly extensible to **Firebase Auth (Google Sign-In) + Cloud Firestore** (`cloud.google.com/go/firestore`).

```text
firesage/
├── Dockerfile                        # Multi-stage: Node Vite build -> Go static binary -> distroless (PORT=8080)
├── go.mod                            # module github.com/ausanyal/firesage
├── cmd/firesage/main.go              # HTTP server entrypoint (0.0.0.0:8080, serves /api/v1/* + embedded SPA)
├── internal/
│   ├── data/                         # Embedded historical datasets (Shiller, Damodaran, NSE/Sensex, RBI, MoSPI) & City profiles
│   ├── tax/                          # India (LTCG/Slab/EPF/PPF/NPS) & USA (Fed+State/Roth Ladder/ACA 2026 Cliff/SSA)
│   ├── engine/                       # Goroutine-parallel Bootstrap, Regime MC, Historical Replay & Goal-Seek Solver
│   ├── store/                        # Extensible PlanRepository interface (ready for Cloud Firestore)
│   └── server/                       # Stateless JSON API handlers + //go:embed SPA static handler
└── web/                              # React 19 + TypeScript + Vite + Tailwind + Interactive Visualization Suite
```
