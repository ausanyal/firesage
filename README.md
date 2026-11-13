# FireSage 🔥

**Global 5-Jurisdiction Institutional Stochastic FIRE (Financial Independence, Retire Early) & Cross-Border Geo-Arbitrage Simulator**

- **Live App:** https://firesage-121012042522.asia-south1.run.app/
- **Supported Jurisdictions:** **India (`₹` INR)** • **United States (`$` USD)** • **Australia (`A$` AUD)** • **Europe & Switzerland (`€` EUR)** • **New Zealand (`NZ$` NZD)**
- **Calibrated Global Cities:** **34 cities & retirement hubs** with **any-to-any cross-border relocation** (accumulate in one country's tax/pension system, retire in any of the 34 cities worldwide).
- **Architecture:** Zero-dependency **Go 1.24** quantitative simulation & tax engine + **React 19 / TypeScript / Vite** UI compiled into both a **single distroless Cloud Run container (`PORT=8080`)** and an **in-browser WebAssembly Worker** for private, local-first execution.

---

## Table of Contents

1. [How to Read & Use the FireSage Dashboard](#1-how-to-read--use-the-firesage-dashboard)
   - [Top Bar & Left Input Rail (Today's Purchasing Power)](#11-top-bar--left-input-rail-all-inputs-in-todays-purchasing-power)
   - [Top Strip — The 5 Executive Verdict Cards](#12-top-strip--the-5-executive-verdict-cards)
   - [Hero Chart — Stochastic Wealth Cone & Trajectory Explorer (`Today's` vs `Future`)](#13-hero-chart--stochastic-wealth-cone--trajectory-explorer)
   - [Middle Left — “When Can I Retire?” Readiness Frontier](#14-middle-left--when-can-i-retire-readiness-frontier)
   - [Middle Right — Lifespan Cash-Flow, Tax & Liquidity Bridge](#15-middle-right--lifespan-cash-flow-tax--liquidity-bridge)
   - [Bottom Card — Goal-Seek Optimizer & Historical Battle-Test Lab](#16-bottom-card--goal-seek-optimizer--historical-battle-test-lab)
2. [How FireSage Calculates Your Required Corpus (Why `25x` / `35x` Breaks Down)](#2-how-firesage-calculates-your-required-corpus-why-25x--35x-breaks-down)
   - [Plain-English Example: Rohan in Goa vs. Priya in Mumbai](#21-plain-english-example-rohan-goa-vs-priya-mumbai)
   - [11-Step Common Random Numbers (CRN) Goal-Seek Solver](#22-11-step-common-random-numbers-crn-goal-seek-solver)
   - [What Pushes Your Personal Multiplier Above or Below `35x`?](#23-what-pushes-your-personal-multiplier-above-or-below-35x)
3. [Unified Mathematical Formulation & Cross-Border FX Engine](#3-unified-mathematical-formulation--cross-border-fx-engine)
4. [Country-by-Country Research, Tax, Pension & Healthcare Models (34 Cities)](#4-country-by-country-research-tax-pension--healthcare-models-34-cities)
   - [🇮🇳 India (`₹` INR — 11 Cities incl. Kolkata, Dehradun, Kodaikanal, Kochi/Cochin)](#41--india--inr--11-cities)
   - [🇺🇸 United States (`$` USD — 7 Cities across 7 State Tax Regimes)](#42--united-states--usd--7-cities)
   - [🇦🇺 Australia (`A$` AUD — 4 Cities)](#43--australia-a-aud--4-cities)
   - [🇪🇺 Europe & Switzerland (`€` EUR — 8 Hubs across 6 Tax Sub-Jurisdictions)](#44--europe--switzerland--eur--8-hubs)
   - [🇳🇿 New Zealand (`NZ$` NZD — 4 Cities)](#45--new-zealand-nz-nzd--4-cities)
5. [Stochastic Simulation Modes & Historical Crisis Cohorts](#5-stochastic-simulation-modes--historical-crisis-cohorts)
6. [Local Development & Google Cloud Run Deployment](#6-local-development--google-cloud-run-deployment)

---

## 1. How to Read & Use the FireSage Dashboard

### 1.1 Top Bar & Left Input Rail (All Inputs in Today's Purchasing Power)

- **Top Header Jurisdiction Selector (`🇮🇳 IN` • `🇺🇸 US` • `🇦🇺 AU` • `🇪🇺 EU/CH` • `🇳🇿 NZ`):**
  Selects your **Working / Accumulation Jurisdiction** — determining your portfolio currency, historical asset/inflation dataset, and statutory retirement account rules during your working years.
- **Section 1 — Timeline & Global Geo-Arbitrage:**
  - **Current Age, Retire Age & Plan Until Age:** Set your current age, candidate retirement age, and longevity horizon (e.g., `90–95`). Numeric inputs buffer while typing so you can erase any box completely blank and type from scratch without premature clamping.
  - **Work City vs. Retire City (Global):** Your Work City comes from your active working country, while **Retire City (Global)** lets you pick **any of the 34 cities across all 5 jurisdictions** (e.g., *Work in Bengaluru → Retire in Kolkata or Goa*, *Work in San Jose → Retire in Portugal or Goa*, *Work in Sydney → Retire in Southern Italy*, or *Work in India → Retire in Raleigh, NC*).
  - **Cross-Border Relocation Banner & `Auto-scale lifestyle by city COL`:** Whenever your Retirement City is in a different city or country, FireSage shows the live FX parity rate, preserves your working country's statutory pension lock-in rules, switches your retirement taxes & healthcare to the destination city on the day you retire, and optionally scales your lifestyle budget to match the destination's Cost-of-Living & Housing index.
- **Section 2 — Corpus, Statutory Retirement Funds & Lifestyle:**
  - **Liquid Spendable Corpus vs. Locked Retirement Funds:** Separated directly in Section 2 so early retirees can distinguish **accessible pre-unlock money** (Mutual Funds / Brokerage / Cash / PIE funds) from **statutorily locked retirement accounts** (`EPF / PPF / NPS` in India, `401(k) / IRA / Roth` in the US, `Superannuation` in Australia, `Pillar 2 / 3a` in Europe/CH, and `KiwiSaver` in New Zealand).
  - **Monthly Liquid Save vs. Monthly Locked Contribution + Step-Up `%`:** Tracks both your taxable monthly investments and your employer/salary-sacrifice pension contributions growing with annual step-ups until retirement.
  - **Target Monthly Spend (Core Living):** Always enter what that lifestyle costs **today** (excluding income taxes and standalone medical insurance, which FireSage models explicitly on top). Use the `Lean` / `Std` / `Chubby` / `Fat` 1-click city presets to load researched local budgets immediately.
- **Sections 2B & 3 (Expandable Accordions):**
  - **Section 2B:** Granular 5-bucket cost-basis & wrapper breakdown, inflation-adjusted State Pensions / Social Security / Rental inflows, Tax-Alpha harvesting toggles, and **One-Time Life Events** (`+` downsizing/inheritance inflows or `-` college/wedding outflows at specific ages).
  - **Section 3:** Asset allocation + **5-Year Retirement Bond Tent**, choice of **4 Stochastic Engines**, **Dynamic Withdrawal Guardrails** (`Guyton–Klinger`, `CAPE Valuation-Dynamic`, or `Fixed Real 4%-Style`), **Retirement Spending Smile**, and **Custom Black-Swan Crash Lab**.

---

### 1.2 Top Strip — The 5 Executive Verdict Cards

Each top card summarizes 3,000 simulated market lifetimes in plain English:

1. **Plan Survival (`%` to Horizon Age):**
   - Percentage of simulated market lifetimes where your total portfolio lasts through `Plan Until Age` (`≥ 90%` emerald = high confidence; `75%–89%` amber = borderline; `< 75%` rose = underfunded).
   - **`Pre-60` / `Pre-65 Liquidity Safe (%)`:** Checks a classic early-FIRE trap — whether your **liquid, unlocked** accounts survive every single year until your locked pension unlocks (`Age 60` in IN/US/AU/EU; `Age 65` for NZ KiwiSaver) without running dry early while millions sit locked behind statutory age gates.
2. **90%-Confidence FIRE Age:**
   - The earliest age you can retire with `≥ 90%` survival odds.
   - **Coast FIRE Age:** The age where your *existing* corpus alone will compound to fund traditional retirement at `60` (`65` in NZ) even if you drop monthly savings to `0` tomorrow.
   - **Fortress FIRE Age:** Ultra-conservative `97%+` survival threshold.
3. **Projected Corpus @ Retire Age:**
   - Your expected median (`P50`) nest egg on the day you retire, plus the unlucky-market (`P10`) to bull-market (`P90`) spread.
4. **90%-Safe Corpus Needed & Implied Initial SWR:**
   - The exact portfolio size needed at your chosen retirement age to achieve `90%` survival, alongside the **Year-1 Safe Withdrawal Rate (`SWR %`)** your total Year-1 draw implies.
5. **Sustainable 90% Spend & Terminal `P50`:**
   - The maximum monthly core budget (in today's buying power) your current trajectory can sustain with `90%` confidence, plus the median legacy estate left over at `Plan Until Age`.

---

### 1.3 Hero Chart — Stochastic Wealth Cone & Trajectory Explorer

- **How to Read the Curves & Right-Margin Pills:**
  - **`P90 Upside` (Top dashed emerald line):** Lucky top-10% bull-market trajectory.
  - **`P50 Median` (Bright solid emerald curve):** Typical middle-of-the-pack wealth path (`50%` of histories finish higher, `50%` lower).
  - **Inner & Outer Shaded Bands:** Span `P25–P75` (middle `50%` of outcomes) and `P10–P90` (`80%` of outcomes).
  - **`P10 Stress` (Bottom rose dashed line):** The 1-in-10 bad sequence of crashes and stagflation. Keeping `P10 Stress` above `0` through your horizon age means your plan has `≥ 90%` survival!
- **`Today's ₹/$ (Real)` vs. `Future ₹/$ (Nominal)` Toggle (Top-Right of Hero Chart):**
  - **`Today's (Real)` *(Recommended for planning)*:** Strips out future inflation so every number on the chart is expressed in **today's grocery-cart buying power**. If the chart says `₹5 Cr` at Age 75 in Real mode, it means wealth that buys what `₹5 Cr` buys *today*.
  - **`Future (Nominal)`:** Raw unadjusted future bank-statement numbers inflated by decades of compounding CPI (useful for seeing actual future account balances, though `₹25 Cr` 35 years from now buys much less than `₹25 Cr` today).
- **Interactive Controls:**
  - **Drag the vertical amber `⇌ FIRE Age` line** left or right directly on the canvas to scrub your retirement age in real time.
  - **Click any trajectory pill below the chart** (`Worst Tail Path`, `P10 Stress Path`, `P50 Median Path`, `P90 Bull Path`) or click a sample spaghetti line on the chart to open a **Year-by-Year Trajectory Autopsy `<dialog>`** showing every single year's sampled historical year, equity/bond returns, CPI, core spend, healthcare bill, taxes paid, liquid wealth, and total real wealth.

---

### 1.4 Middle Left — “When Can I Retire?” Readiness Frontier

Plots your **Plan Survival Probability (`0% → 100%`)** across every candidate retirement age:
- **Solid Emerald Curve:** Survival probability at each retirement age under your active plan (including relocation if selected).
- **Gray Dotted Curve (`Baseline if Staying in <Work City>`):** Appears automatically whenever `Work City ≠ Retire City` so you can visually compare your survival odds if you *stayed* in your expensive work metro vs. relocating.
- **Horizontal Green Dashed Line (`90% Safe Target`):** Marks the institutional `90%` confidence threshold — where the solid emerald curve crosses this dashed line is your **Target FIRE Age**.
- **Vertical Amber Dashed Line (`Your Plan: Age X`):** Highlights your currently selected retirement age. Click any point on the curve or any milestone badge (`⛵ Coast`, `🌱 Lean`, `🎯 Target`, `🏰 Fortress`) to jump to that age.

---

### 1.5 Middle Right — Lifespan Cash-Flow, Tax & Liquidity Bridge

Contains a 2-way toggle solving two distinct retirement failure modes:

1. **`Spend + Health + Tax` View *(Why it exists: Exposing Hidden Retirement Costs)*:**
   Your monthly lifestyle budget is only part of your retirement bill. This stacked area chart reveals how **Core Living Spend** (green), **Compounding Healthcare** (blue — e.g., `11%–13%` Indian medical CPI, the US `2026 400% FPL ACA Cliff → Age 65 Medicare`, Swiss `LAMal`, or age-rated NZ Southern Cross surgical premiums), and **Income / Capital Gains / Wealth Taxes** (amber) evolve every year, offset by **Guaranteed Pensions / Social Security / Age Pension / NZ Super** (purple).
2. **`Liquid vs Locked` View *(Why it exists: Preventing the Pre-60 / Pre-65 Liquidity Trap)*:**
   If you retire at Age `42`, you cannot touch `NPS`, `401(k)`, `Superannuation`, or `KiwiSaver` penalty-free for `18–23` years. This view plots **Spendable Liquid Wealth** (emerald) against **Locked Retirement Accounts** (indigo) with explicit vertical markers at **`FIRE <Age>`** and **`Age 60 Unlock`** (`Age 65 Unlock` for New Zealand), alerting you if liquid funds risk running dry before unlock day.

---

### 1.6 Bottom Card — Goal-Seek Optimizer & Historical Battle-Test Lab

- **`🚀 Action Levers`:** Ranked 1-click optimizations (*Boost Savings `+25%`*, *Delay Retirement `+2 Yrs`*, *5-Year Consulting/Barista Bridge*, *Trim Spend `-12%`*, *Enable Dynamic Guardrails + Bond Tent*, *Tier-2 Geo-Arbitrage*) showing exact `+Survival %` delta and `-Years Saved to FIRE`. Click **`Apply Lever`** to apply any move immediately.
- **`🏛️ Historical Crises`:** Replays your exact plan as if you retired on the eve of each jurisdiction's most notorious market/inflation storms (e.g., *1929 Great Depression*, *1966 US Stagflation*, *1992 Harshad Mehta + High CPI*, *2008 GFC*, *1973 Aussie Stagflation*, *1987 NZX Crash (`-60%`)*, *2011 Eurozone Sovereign Debt Crisis*).
- **`🛡️ Spending Guardrails`:** Plots your real monthly spending band (`P90` prosperity raises, `P50` median budget, and `P10` bear-market guardrail floor) across retirement.
- **`🔥 Sequence Risk Matrix`:** A `5×4` survival heatmap crossing **First-5-Year Real Portfolio Return** against **First-5-Year Inflation Regime** right after retiring — showing at a glance whether early crashes, high inflation, or stagflation pose the sharpest risk.

---

## 2. How FireSage Calculates Your Required Corpus (Why `25x` / `35x` Breaks Down)

Popular rules of thumb multiply annual spending by **`25x`** (the 1994 US `4%` Trinity Study for a 30-year retirement ignoring taxes and medical inflation) or **`33x–35x`** (`~2.85%–3.0%` SWR used in early-FIRE communities). **FireSage never uses a fixed multiplier.**

### 2.1 Plain-English Example: Rohan (Goa) vs. Priya (Mumbai)

Imagine **Rohan** and **Priya** both spend **`₹1 Lakh/month` (`₹12 Lakhs/year`)** today. A naive `35x` calculator tells *both* of them they need **`₹4.2 Crore`**. Look at why that fails:

| Dimension | Rohan (Age 52, moving to Goa) | Priya (Age 38, staying in Mumbai) |
| :--- | :--- | :--- |
| **Retirement Horizon** | **38 years** (`Age 52 → 90`) | **52 years** (`Age 38 → 90`) — 14 extra years of CPI & medical compounding |
| **City Living Costs** | Moving to Goa cuts living & housing bills by **~22%–35%** | Pays Zone-1 Mumbai housing & hospital prices forever |
| **Backup Income** | Pension / rental income kicks in at Age `60` | Zero pension — portfolio pays `100%` of bills for 52 years |
| **Crash Flexibility** | Happy to skip a foreign holiday if stocks drop `-35%` | Rigid fixed spending even at the bottom of a market crash |
| **True 90%-Safe Corpus** | **`~₹2.7–₹3.0 Cr` (`~23x–25x`)** — `35x` traps him at work unnecessarily! | **`~₹5.0–₹5.5 Cr` (`~42x–46x`)** — retiring on `35x` risks depletion in her 70s/80s! |

> Saying *"everyone needs 35x"* is like saying *"every road trip needs 35 liters of fuel"* without asking **how far you are driving**, **whether the road goes uphill or downhill**, or **whether you can refuel along the way**.

### 2.2 11-Step Common Random Numbers (CRN) Goal-Seek Solver

In [`internal/engine/solver.go`](internal/engine/solver.go) (`solveRequiredCorpusAtRetireCRN`), FireSage runs an **11-iteration binary search** across `[4x, 62x]` of your COL-scaled annual spending using **Glasserman Common Random Numbers (CRN)** variance reduction:
1. Fast-forwards your plan to your chosen `Retirement Age` across the exact same 3,000 multi-decade market/inflation scenarios.
2. Simulates every year of decumulation (core spend + residency healthcare + residency taxes − state/private pensions + life events + guardrail rules) through `Plan Until Age`.
3. Halves the search interval 11 times (`1 / 2^11 ≈ 0.05%` precision) to find the exact nest egg where **90.0% of market histories survive**.

### 2.3 What Pushes Your Personal Multiplier Above or Below `35x`?

| Factor Modeled in FireSage | Typical Impact vs `35x` | Reason |
| :--- | :--- | :--- |
| **Standalone Healthcare Inflation** | **Pushes UP** (`+4x` to `+10x`) | Healthcare is modeled *on top* of core spend and compounds faster than general CPI. |
| **Withdrawal, Wealth & FIF Taxes** | **Pushes UP** (`+2x` to `+7x`) | Gross portfolio sales must cover capital gains, ordinary slab/RMD taxes, Swiss wealth tax, or NZ FIF drag. |
| **Ultra-Early Horizon (`Age 35–42 → 95`)** | **Pushes UP** (`+4x` to `+8x`) | Surviving 50+ years requires lower initial drawdowns than a 30-year retirement. |
| **Domestic or Cross-Border Geo-Arbitrage** | **Pulls DOWN** (`-5x` to `-18x`) | Moving from San Jose → Algarve or Bengaluru → Kolkata/Mysuru slashes ongoing real outflows. |
| **Dynamic Guardrails & Spending Smile** | **Pulls DOWN** (`-4x` to `-8x`) | Trimming discretionary spend slightly in severe drawdowns prevents selling equities at crash lows. |
| **State Pensions (Age Pension / NZ Super / SS)** | **Pulls DOWN** (`-4x` to `-12x`) | Guaranteed government or private pensions at `60` / `65` / `67` shrink late-life portfolio withdrawals. |

---

## 3. Unified Mathematical Formulation & Cross-Border FX Engine

For any working country $w \in \{\text{IN}, \text{US}, \text{AU}, \text{EU}, \text{NZ}\}$ and any global retirement city $c_r$ (in country $r$), the required gross portfolio withdrawal at age $a \ge A_{\text{retire}}$ in working-currency terms is:

$$
D_{\text{gross}}(a) = \max\!\Big(0,\; E_{\text{core}}(a)\cdot S_{\text{COL}}(c_w \to c_r) + H_{\text{med}}(a, c_r)\cdot FX_{r \to w} + T_{\text{res}}(a, c_r)\cdot FX_{r \to w} - P_{\text{guar}}(a) - M(a)\Big)
$$

where the category-weighted Cost-of-Living & Housing relocation factor is:

$$
S_{\text{COL}}(c_w \to c_r) = 0.65 \cdot \frac{\text{COL}(c_r)}{\text{COL}(c_w)} + 0.35 \cdot \frac{\text{Housing}(c_r)}{\text{Housing}(c_w)}
$$

- **Reference Long-Run FX Parity (`internal/data/cities.go`):** `1.00 USD = ₹85.00 INR = A$1.46 AUD = €0.94 EUR = NZ$1.65 NZD`.
- **Cross-Border Account & Residency Hand-Off:**
  - **Before Retirement (`a < A_retire`):** Contributions, employer matches, and internal pension drag follow your **Working Jurisdiction ($w$)**.
  - **Statutory Unlock Age ($A_{\text{unlock}}$):** Follows the legal rules of the country where your locked accounts are held (`Age 65` for New Zealand KiwiSaver; `Age 60` for India EPF/PPF/NPS, US 401(k)/IRA, Australian Superannuation, and European Pillar 2/3a).
  - **During Retirement (`a >= A_retire`):** Living costs, medical insurance/OOP inflation, and tax residency rules (`T_res` & `H_med`) automatically switch to your **Retirement City ($c_r$)**.

---

## 4. Country-by-Country Research, Tax, Pension & Healthcare Models (34 Cities)

### 4.1 🇮🇳 India (`₹` INR — 11 Cities)

- **Historical Dataset (`1980–2025`):** S&P BSE Sensex Total Return spliced with NSE Nifty 50 TRI, RBI 10-Year GoI G-Sec & 91-Day T-Bill yields, INR Gold, MoSPI CPI-Combined, and `11%–14%` Indian private medical inflation.
- **Tax & Lock-In Mechanics (`internal/tax/india.go`):**
  - **FY 2025–26 New Tax Regime:** `0%` up to `₹4L`, `5%` (`₹4L–8L`), `10%` (`₹8L–12L`), `15%` (`₹12L–16L`), `20%` (`₹16L–20L`), `25%` (`₹20L–24L`), `30%` (`>₹24L`) + **Section 87A zero-tax rebate up to `₹12L`** on slab income (properly *excluding* Section 112A special-rate equity gains) + `4%` Health & Education Cess.
  - **Equity & Debt Withdrawals:** `12.5%` Section 112A Equity LTCG above `₹1.25L/yr` tax-free harvesting exemption; Debt Mutual Funds taxed at slab rates.
  - **Locked Corpus (`Age 60` Unlock):** `EPF / VPF`, `PPF` (EEE tax-free), and **NPS `60%` tax-free lump sum + `40%` mandatory annuity** at Age `60` (`25%` early-exit liquidity cap before 60).
- **Healthcare Formula:** Zone-calibrated retail **Family Floater + `₹1 Cr` Super Top-Up** base premium escalated by age bands (`1.0x` `<45` → `3.8x` `75+`) compounding at Indian Medical CPI, plus an annual **Tertiary Medical Travel Buffer** for non-metro / hill-station towns.

| City (`ID`) | Region | Tier | COL / Housing Index | Monthly Couple Presets (`Lean` / **`Std`** / `Chubby` / `Fat`) | Typical 3BHK Home | Tertiary Medical Buffer |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Bengaluru** (`in-bengaluru`) | Karnataka | Tier 1 | `1.00` / `1.00` | `₹75K` / **`₹1.40L`** / `₹2.60L` / `₹5.00L` | `₹2.25 Cr` | `₹0` |
| **Mumbai** (`in-mumbai`) | Maharashtra | Tier 1 | `1.24` / `1.60` | `₹1.05L` / **`₹1.85L`** / `₹3.40L` / `₹6.50L` | `₹3.80 Cr` | `₹0` |
| **Delhi-NCR** (`in-delhi-ncr`) | NCR | Tier 1 | `1.06` / `1.15` | `₹82K` / **`₹1.50L`** / `₹2.80L` / `₹5.40L` | `₹2.60 Cr` | `₹0` |
| **Hyderabad** (`in-hyderabad`) | Telangana | Tier 1 | `0.94` / `0.92` | `₹70K` / **`₹1.30L`** / `₹2.40L` / `₹4.60L` | `₹1.95 Cr` | `₹0` |
| **Kolkata** (`in-kolkata`) | West Bengal | Tier 1 | `0.76` / `0.62` | `₹55K` / **`₹1.05L`** / `₹1.95L` / `₹3.70L` | `₹1.35 Cr` | `₹0` *(Apollo / Medica / Fortis)* |
| **Goa** (`in-goa`) | Goa | Tier 2 | `0.78` / `0.65` | `₹58K` / **`₹1.08L`** / `₹2.00L` / `₹3.90L` | `₹1.45 Cr` | `₹60K/yr` |
| **Mysuru** (`in-mysuru`) | Karnataka | Tier 2 | `0.68` / `0.48` | `₹48K` / **`₹92K`** / `₹1.70L` / `₹3.20L` | `₹1.05 Cr` | `₹35K/yr` |
| **Coimbatore** (`in-coimbatore`) | Tamil Nadu | Tier 2 | `0.70` / `0.50` | `₹50K` / **`₹95K`** / `₹1.75L` / `₹3.30L` | `₹1.10 Cr` | `₹20K/yr` |
| **Kochi (Cochin)** (`in-kochi`) | Kerala | Tier 2 | `0.74` / `0.55` | `₹54K` / **`₹1.02L`** / `₹1.88L` / `₹3.55L` | `₹1.20 Cr` | `₹15K/yr` *(Aster / Amrita)* |
| **Dehradun** (`in-dehradun`) | Uttarakhand | Tier 2 | `0.72` / `0.56` | `₹52K` / **`₹98K`** / `₹1.80L` / `₹3.40L` | `₹1.45 Cr` | `₹18K/yr` *(Max / AIIMS Rishikesh)* |
| **Kodaikanal** (`in-kodaikanal`) | Tamil Nadu | Tier 2 | `0.71` / `0.58` | `₹50K` / **`₹96K`** / `₹1.76L` / `₹3.35L` | `₹1.65 Cr` | `₹55K/yr` *(Madurai ~3 hrs)* |

---

### 4.2 🇺🇸 United States (`$` USD — 7 Cities)

- **Historical Dataset (`1928–2025`):** NYU Stern (Damodaran) S&P 500 Total Return, 10Y US Treasury, 3M T-Bill & Gold + Robert Shiller CAPE (`P/E10`) & BLS Headline/Medical CPI.
- **Tax, Roth Ladder & RMD Engine (`internal/tax/usa.go`):**
  - **Federal + State Taxes:** Ordinary Federal brackets (`10%` to `37%` after `$30k` MFJ / `$15k` Single standard deduction), stacked `0% / 15% / 20%` Federal LTCG + `3.8%` NIIT (`>$250k`/`$200k`), `85%` max Social Security taxability, and state modules for `CA` (`13.3%`), `NY` (`14.8%` w/ NYC), `NC` (`4.25%` flat), `CO` (`4.40%` flat), `WA` (`0%` ordinary / `7%` high-LTCG excise), and `TX`/`FL` (`0%`).
  - **ACA-Safe 5-Year Roth Conversion Ladder & RMDs:** Automatically converts Traditional `401(k)/IRA` balances into a 5-year seasoning queue capped strictly beneath the restored **2026 hard `400%` FPL ACA Subsidy Cliff**, plus SECURE 2.0 **Age `73`/`75` RMDs** (and `10%` Section 72(t) early withdrawal penalty if unseasoned locked funds are breached pre-`59.5`).
- **Healthcare Formula:** Pre-65 **ACA Marketplace Benchmark Silver** (`8.5%` MAGI cap below `400%` FPL vs. full unsubsidized age-rated cliff above `400%` FPL) transitioning at **Age `65`** to **Medicare Part B + Part D + Medigap Plan G + IRMAA** high-income surcharges.

| City (`ID`) | State | Tier | COL / Housing Index | Monthly Couple Presets (`Lean` / **`Std`** / `Chubby` / `Fat`) | Typical Home | Top State Tax |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **San Jose / Silicon Valley** (`us-san-jose`) | `CA` | Tier 1 | `1.58` / `2.15` | `$6,200` / **`$10,500`** / `$17,500` / `$30,000` | `$1.75M` | `13.3%` |
| **New York City** (`us-nyc`) | `NY` | Tier 1 | `1.62` / `2.25` | `$6,500` / **`$11,000`** / `$18,500` / `$32,000` | `$1.65M` | `14.8%` |
| **Seattle / Eastside** (`us-seattle`) | `WA` | Tier 1 | `1.32` / `1.60` | `$5,200` / **`$8,800`** / `$14,800` / `$25,500` | `$1.15M` | `0%` ord (`7%` high LTCG) |
| **Raleigh / Research Triangle** (`us-raleigh`) | `NC` | Tier 2 | `0.96` / `0.88` | `$3,900` / **`$6,500`** / `$11,000` / `$19,000` | `$540K` | `4.25%` flat |
| **Austin** (`us-austin`) | `TX` | Tier 2 | `1.02` / `0.95` | `$4,100` / **`$6,900`** / `$11,600` / `$20,000` | `$580K` | `0%` |
| **Tampa / Sarasota** (`us-tampa`) | `FL` | Tier 2 | `0.98` / `0.92` | `$4,000` / **`$6,600`** / `$11,200` / `$19,500` | `$510K` | `0%` |
| **Denver / Front Range** (`us-denver`) | `CO` | Tier 2 | `1.10` / `1.15` | `$4,400` / **`$7,400`** / `$12,500` / `$21,500` | `$660K` | `4.40%` flat |

---

### 4.3 🇦🇺 Australia (`A$` AUD — 4 Cities)

- **Historical Dataset (`1980–2025`):** S&P/ASX 200 Accumulation Index (including franked dividend yield), RBA 10-Year Commonwealth Govt Bonds & Cash Rate, AUD Gold, ABS CPI & AIHW Health CPI (`internal/data/intl_history.go`).
- **Superannuation, Tax & Age Pension Formula (`internal/tax/australia.go`):**
  - **Superannuation (`Preservation Age 60`):** Concessional Super contributions taxed at **`15%`** on entry (`85%` net invested), `~0.6%` internal accumulation fund tax drag before 60, and **100% tax-free Account-Based Pension withdrawals & earnings (`0%` tax) from Age `60`**, subject to statutory minimum pension drawdown rates (`4%` ages 60–64, `5%` 65–74, `6%` 75–79, `7%` 80–84, `9%` 85–89, `11%` 90–94, `14%` 95+).
  - **Personal Brokerage (FY 2025–26 Stage 3 Brackets):** `0%` up to `A$18.2k`, `16%` (`A$18.2k–45k`), `30%` (`A$45k–135k`), `37%` (`A$135k–190k`), `45%` (`>A$190k`) + Low Income Tax Offset (`LITO` up to `A$700`) + `2%` Medicare Levy + **`50%` long-term CGT discount** + **ASX `30%` franking credit imputation offset**.
  - **Centrelink Age Pension (Age `67+`):** Dual statutory homeowner **Assets Test + Deeming Income Test** — full pension including supplements (`A$48,516/yr` couple / `A$32,214/yr` single) takes the stricter of the Assets-Test taper (**`A$78/yr` per `A$1,000`** above `A$470,000` couple / `A$314,000` single) and the Income-Test taper (**`50c` per `A$1`** of deemed financial + private income above `A$9,672/yr` couple / `A$5,512/yr` single), combining additively with private pensions.
- **Healthcare Formula:** Universal Medicare + Private Hospital Silver/Gold & Extras cover (`~A$5,200/yr` couple, avoiding the `1%–1.5%` Medicare Levy Surcharge) + specialist/dental gap out-of-pocket costs escalating with age.

| City (`ID`) | State | Tier | COL / Housing Index | Monthly Couple Presets (`Lean` / **`Std`** / `Chubby` / `Fat`) | Typical Home |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Sydney** (`au-sydney`) | `NSW` | Tier 1 | `1.08` / `1.28` | `A$4,600` / **`A$6,800`** / `A$11,200` / `A$19,000` | `A$1.65M` |
| **Melbourne** (`au-melbourne`) | `VIC` | Tier 1 | `1.00` / `1.00` | `A$4,100` / **`A$6,100`** / `A$10,200` / `A$17,500` | `A$1.12M` |
| **Gold Coast / Sunshine Coast** (`au-gold-coast`) | `QLD` | Tier 2 | `0.90` / `0.88` | `A$3,700` / **`A$5,500`** / `A$9,200` / `A$15,800` | `A$1.05M` |
| **Adelaide** (`au-adelaide`) | `SA` | Tier 2 | `0.85` / `0.78` | `A$3,500` / **`A$5,200`** / `A$8,600` / `A$14,800` | `A$880K` |

---

### 4.4 🇪🇺 Europe & Switzerland (`€` EUR — 8 Hubs)

- **Historical Dataset (`1980–2025`):** MSCI Europe / STOXX 600 Net Total Return, German 10Y Bunds & 3M Euribor, EUR Gold, and Eurostat HICP / Swiss BFS CPI.
- **6 Distinct Sub-Jurisdiction Tax & Healthcare Engines (`internal/tax/europe.go`):**
  1. **Switzerland — Zurich Suburbs (`CH-ZH`) & Lugano / Lake Maggiore (`CH-TI`):** **`0%` private capital gains tax** on movable securities + annual **Cantonal & Municipal Net Wealth Tax** (`~0.18%–0.28%` in Zurich suburbs; `~0.24%–0.34%` in Ticino above tax-free threshold) + separate `~5.5%–6.5%` capital withdrawal tax on Pillar 2 (`BVG`) / Pillar 3a + mandatory per-capita **`LAMal` / `KVG` health insurance** (`~€510–€545/mo` per adult + deductible & `10%` retention).
  2. **Southern Italy — Puglia & Sicily (`IT-SOUTH`):** Models **Article 24-ter `7%` Flat Foreign Retiree Tax** on foreign-source investment & pension income for the **first 10 years of retirement** in eligible `<20k`-population Mezzogiorno towns, automatically reverting in Year `11+` to Italy's `26%` CGT + `0.20%` `IVAFE` wealth duty, plus `SSN` (`€2,000/yr` initial buy-in) + private hospital cover.
  3. **Greece — Chania / Crete & Peloponnese (`GR`):** Models **Article 5B `7%` Flat Foreign Retiree Tax** for the **first 15 years** of retirement, reverting to `15%` UCITS/listed equity gains thereafter, plus `EFKA` + private hospital cover.
  4. **Portugal — Algarve & Cascais / Lisbon Coast (`PT`):** Models **Law 31/2024** progressive capital gains holding exclusion (`30%` exclusion after `8+` years → **`19.6%` effective CGT** vs. `28%` standard autonomous rate; `0%` wealth tax on securities) + `SNS` public access + age-rated `Médis`/`Multicare` private health cover.
  5. **Spain — Valencia & Costa del Sol / Málaga (`ES`):** Progressive *Renta del Ahorro* savings brackets (`19%` to `€6k`, `21%` to `€50k`, `23%` to `€200k`, `27%` to `€300k`, `30%` `>€300k`) + regional wealth check + `Sanitas`/`Adeslas` no-copay private insurance.
  6. **France — Nice & Aix-en-Provence (`FR`):** `30%` *PFU* (*Prélèvement Forfaitaire Unique*, reduced to `17.2%` social charges inside seasoned `PEA`) + **`6.5%` PUMA *Cotisation Subsidiaire Maladie* (`CSM`)** passive-income healthcare levy above `€23,500` (waived once pension income starts) + *Assurance Maladie* & *Mutuelle*.

| Hub (`ID`) | Sub-Regime | Tier | COL / Housing Index | Monthly Couple Presets (`Lean` / **`Std`** / `Chubby` / `Fat`) | Typical Home | Key Tax Feature |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Zurich Suburbs — Winterthur / Baden / Thalwil** (`eu-zurich-suburbs`) | `CH-ZH` | Tier 1 | `1.55` / `1.70` | `€3,900` / **`€5,600`** / `€9,200` / `€16,000` | `€1.35M` | `0%` CGT + low ZH wealth tax |
| **Lugano & Lake Maggiore Suburbs** (`eu-lugano`) | `CH-TI` | Tier 1 | `1.42` / `1.45` | `€3,500` / **`€5,100`** / `€8,500` / `€14,800` | `€1.15M` | `0%` CGT + Swiss lake belt |
| **Cascais & Lisbon Coast** (`eu-cascais-lisbon`) | `PT` | Tier 1 | `0.92` / `0.96` | `€2,350` / **`€3,400`** / `€5,800` / `€10,200` | `€680K` | `19.6%` (`8yr+`) – `28%` CGT |
| **Algarve — Lagos / Tavira / Faro** (`eu-algarve`) | `PT` | Tier 2 | `0.78` / `0.72` | `€1,950` / **`€2,850`** / `€4,800` / `€8,500` | `€460K` | `19.6%` (`8yr+`) CGT, `0%` wealth tax |
| **Southern Italy — Puglia & Sicily** (`eu-south-italy`) | `IT-SOUTH` | Tier 2 | `0.62` / `0.48` | `€1,550` / **`€2,250`** / `€3,800` / `€6,800` | `€260K` | **`7%` Flat Tax (10 yrs)** |
| **Valencia & Costa del Sol / Málaga** (`eu-valencia`) | `ES` | Tier 2 | `0.80` / `0.74` | `€2,000` / **`€2,950`** / `€4,900` / `€8,800` | `€410K` | `19%–30%` *Renta del Ahorro* |
| **Nice & Aix-en-Provence** (`eu-nice-provence`) | `FR` | Tier 2 | `1.00` / `1.02` | `€2,500` / **`€3,650`** / `€6,200` / `€11,000` | `€590K` | `30%` PFU (`17.2%` PEA) + PUMA |
| **Chania (Crete) & Peloponnese** (`eu-crete-greece`) | `GR` | Tier 2 | `0.65` / `0.52` | `€1,600` / **`€2,350`** / `€3,950` / `€7,100` | `€310K` | **`7%` Flat Tax (15 yrs)** |

---

### 4.5 🇳🇿 New Zealand (`NZ$` NZD — 4 Cities)

- **Historical Dataset (`1980–2025`):** S&P/NZX 50 Gross Index, RBNZ 10Y NZGB & 90-Day Bank Bills, NZD Gold, Stats NZ CPI & private surgical medical inflation.
- **KiwiSaver, FIF `5%` FDR & Universal NZ Super Formula (`internal/tax/newzealand.go`):**
  - **KiwiSaver (`Age 65` Lock-In):** Strictly preserved until **Age `65`** (automatically setting `Pre-65 Liquidity Safe` and `Age 65 Unlock` across the app), unlocking **100% tax-free** at Age `65`.
  - **Universal NZ Superannuation (Age `65+`):** **Non-means-tested** (`0%` asset or income clawback!), providing `~NZ$42,000/yr` net (`M` code) for a qualifying couple or `~NZ$27,500/yr` for a single retiree.
  - **`0%` General CGT + Foreign Investment Fund (`FIF`) `5%` FDR Drag:** Domestic/Australasian shares carry `0%` general capital gains tax; global shares above the `NZ$50k` (`NZ$100k` couple) *de minimis* cost threshold are taxed on a **`5%` deemed Fair Dividend Rate (`FDR`)** return capped at the **`28%` Multi-Rate PIE PIR** (`~1.40%` max annual drag on global shares). Incoming cross-border retirees automatically receive New Zealand's **4-year Transitional Tax Resident `0%` FIF holiday**.
- **Healthcare Formula:** Public Health NZ + universal ACC injury cover + **Southern Cross private surgical insurance**, capturing NZ's steep age-rated surgical premium curve (`1.00x` at 55 → `1.65x` at 65 → `2.55x` at 75 → `3.20x` at 80+).

| City (`ID`) | Region | Tier | COL / Housing Index | Monthly Couple Presets (`Lean` / **`Std`** / `Chubby` / `Fat`) | Typical Home |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Auckland** (`nz-auckland`) | North Island | Tier 1 | `1.00` / `1.05` | `NZ$4,200` / **`NZ$6,000`** / `NZ$9,800` / `NZ$16,800` | `NZ$1.08M` |
| **Wellington** (`nz-wellington`) | North Island | Tier 1 | `0.95` / `0.92` | `NZ$3,950` / **`NZ$5,700`** / `NZ$9,300` / `NZ$15,900` | `NZ$890K` |
| **Tauranga / Nelson** (`nz-tauranga-nelson`) | Coastal NZ | Tier 2 | `0.90` / `0.88` | `NZ$3,750` / **`NZ$5,400`** / `NZ$8,800` / `NZ$15,000` | `NZ$920K` |
| **Queenstown / Wanaka** (`nz-queenstown`) | Otago Alps | Tier 2 | `1.12` / `1.25` | `NZ$4,800` / **`NZ$6,800`** / `NZ$11,400` / `NZ$19,500` | `NZ$1.55M` |

---

## 5. Stochastic Simulation Modes & Historical Crisis Cohorts

1. **Politis & Romano (1994) Stationary Block Bootstrap *(Default)*:** Samples random-length geometric blocks (mean `4` years) of real historical years so multi-year stagflation waves, fat-tail crashes, and cross-asset correlations remain intact without Gaussian IID assumptions.
2. **Valuation / CAPE Regime-Switching + Merton Jump-Diffusion Monte Carlo:** Conditions 10–15 year forward equity drift on starting valuation multiples (`Shiller CAPE`, `Nifty P/E`, `ASX P/E`, `STOXX P/E`, `NZX P/E`) with Poisson crash jumps.
3. **Historical Rolling Cohorts & Named Crisis Replays:** Deterministically tests every rolling historical window plus famous stress cohorts across all 5 jurisdictions.
4. **Custom Black-Swan Crash Lab:** Inject deterministic Year-1 retirement equity crashes (`-15%` to `-60%`), multi-year inflation spikes, and lump-sum medical emergencies at exact ages.

---

## 6. Local Development & Google Cloud Run Deployment

### Prerequisites
- **Go 1.24+**
- **Node.js 22+ & npm**

### Build, Test & Preview Locally
```bash
# 1. Run Go quantitative, cross-border & 5-jurisdiction tax unit tests
make test

# 2. Compile Go WebAssembly worker (web/public/firesage.wasm) + production Vite bundle
make web

# 3. Preview locally on http://127.0.0.1:8080
npm --prefix web run preview
```

Or build and run the single native Go server binary (serves `/api/v1/*` concurrent goroutine endpoints + embedded SPA on `PORT=8080`):
```bash
make build
PORT=8080 ./firesage
```

### Continuous Deployment on Google Cloud Run
FireSage ships with a multi-stage, zero-dependency distroless root [`Dockerfile`](Dockerfile) exposing `PORT=8080`:
- Pushing to `main` on `github.com/ausanyal/firesage` automatically triggers **Google Cloud Build** and rolls out a new revision to **Cloud Run (`asia-south1`)** at https://firesage-121012042522.asia-south1.run.app/.
- Manual one-command deploy via `gcloud`:
  ```bash
  gcloud run deploy firesage \
    --source . \
    --region asia-south1 \
    --allow-unauthenticated \
    --port 8080 \
    --memory 512Mi \
    --cpu 2
  ```

### Additional Architecture Documentation
See [`docs/ARCHITECTURE_AND_DESIGN.md`](docs/ARCHITECTURE_AND_DESIGN.md) for the low-level systems architecture and local-first `IndexedDB` / `PlanRepository` schema.
