# FireSage 🔥

**Institutional-Grade Stochastic FIRE (Financial Independence, Retire Early) & Lifecycle Simulation Platform for India (`₹`) and the USA (`$`).**

Built with a **zero-dependency Go quantitative backend** + **React 19 / TypeScript / Vite** interactive visualization suite compiled into both a **single static container (`PORT=8080`) for Google Cloud Run** and a **100% private in-browser WebAssembly Worker** for zero-latency offline/local-first execution.

---

## Highlights & Capabilities

### 1. Native Dual-Jurisdiction Engines (`🇮🇳 India` & `🇺🇸 USA`)
- **India (`₹ Cr / ₹ L`):**
  - **Historical Series (1980–2025):** BSE Sensex Total Return / Nifty 50 TRI, RBI 10-Year G-Sec & 91-Day T-Bill yields, INR Gold, and MoSPI CPI-IW / Combined Headline & Medical CPI (`~12%` healthcare inflation).
  - **Tax & Statutory Lock-In Modeling:** FY 2025–26 New Tax Regime slabs (`₹12L` Sec 87A zero-tax threshold + marginal relief), Sec 112A `12.5%` Equity LTCG (`₹1.25L` annual exemption) + `4%` cess, Cost-Basis tracking, EPF/PPF vs. **NPS Age-60 `60%/40%` lump-sum/annuity unlock**, and **Pre-60 Liquidity Bridge Alerts**.
  - **Healthcare Shield:** Age-banded private Family Floater + Super Top-Up premiums compounding at dedicated medical inflation rates.
- **USA (`$M / $K`):**
  - **Historical Series (1928–2025):** NYU Stern (Damodaran) & Robert Shiller S&P 500 Total Return, 10-Year US Treasury, 3-Month T-Bill, Gold, BLS CPI-U, Medical CPI, and cyclically adjusted **Shiller CAPE (`36.2x`)**.
  - **Tax, Roth Ladder & RMD Engine:** Federal ordinary brackets + stacked `0% / 15% / 20%` LTCG + `3.8%` NIIT, state tax rules (`CA`, `NY`, `NC`, `WA`, `CO`, `TX`, `FL`), **5-Year Roth Conversion Ladder** seasoning queue, and SECURE 2.0 Age `73` RMDs.
  - **2026 ACA Subsidy Cliff → Medicare Bridge:** Explicit post-2025 `400%` FPL hard subsidy cliff check during early retirement (`Age < 65`), ACA-safe Roth conversion headroom, and automatic transition to **Medicare Part B + Part D + Medigap + IRMAA** at `Age 65`.

### 2. Domestic Tier-1 → Tier-2 Geo-Arbitrage
- **India:** Compare working in Tier-1 metros (*Bengaluru, Mumbai, Delhi-NCR, Hyderabad*) and retiring in Tier-2 lifestyle hubs (*Goa, Mysuru, Coimbatore, Kochi*) with category-weighted housing/services/healthcare multipliers.
- **USA:** Compare accumulating in VHCOL hubs (*San Jose / Silicon Valley CA, New York City NY, Seattle WA*) and retiring in Tier-2 MCOL/tax-friendly hubs (*Raleigh NC, Austin TX, Tampa FL, Denver CO*) with simultaneous Cost-of-Living and state income/capital-gains tax arbitrage.

### 3. 4-Mode Quantitative Simulation Core
1. **Politis–Romano Stationary Block Bootstrap** (preserves multi-year inflation waves, equity-bond correlation spikes, and mean reversion).
2. **Valuation / CAPE Regime-Switching + Merton Jump-Diffusion Monte Carlo** (conditions forward 10–15 year expected equity drift on starting Shiller CAPE / Nifty P/E).
3. **Historical Cohort & Named Crisis Replay** (*1929 Great Depression, 1966 Stagflation, 2000 Dot-Com + GFC Lost Decade, 1992 Harshad Mehta + High CPI, 2008 Nifty Crash, 2010–2013 Indian Stagflation*).
4. **Custom Black-Swan Crash Lab** (inject deterministic `-20%` to `-60%` equity drawdowns and stagflation spikes at exact retirement ages).

### 4. Interactive Visualization & Goal-Seek Suite
- **Stochastic Wealth Cone & Trajectory Explorer:** Real (`Today's ₹/$`) vs. Nominal toggle, `P10–P90` fan bands, **draggable `⇌ FIRE Age` vertical scrub line**, and clickable sample paths opening a **Year-by-Year Trajectory Autopsy `<dialog>`**.
- **“When Can I Retire?” Readiness Frontier:** Survival probability across every candidate retirement age (`Coast FIRE`, `Lean FIRE`, `Target 90% FIRE`, `Fortress 97% FIRE`) + Geo-Arbitrage uplift curve.
- **Lifespan Cash-Flow, Tax & Liquidity Bridge:** Stacked streamgraph of Core Living, Healthcare, Taxes, and Pension/Social Security inflows vs. Liquid vs. Locked retirement buckets.
- **1-Click Action Levers, Dynamic Guardrail Corridors & Sequence-of-Returns Risk (SRR) 5×4 Matrix.**
- **100% Local-First Persistence:** Zero-signup `IndexedDB` plan vault + JSON export/import + shareable `#plan=...` URL snapshots, architected behind a pluggable `PlanRepository` interface (`internal/store/repository.go` & `web/src/storage/planRepository.ts`) ready for **Google Sign-In + Cloud Firestore** sync.

---

## Quick Start (Local Development)

### Prerequisites
- **Go 1.24+**
- **Node.js 22+ & npm**

### Build & Run
```bash
# 1. Run Go quantitative & tax/healthcare unit tests
make test

# 2. Compile Go WASM worker + production React/Vite bundle
make web

# 3. Preview locally on http://127.0.0.1:8080 (runs via embedded Go API or in-browser Go WASM worker)
npm --prefix web run preview
```

Or build and run the single native Go binary (serves both `/api/v1/*` concurrent goroutine endpoints and embedded SPA on `http://0.0.0.0:8080`):
```bash
make build
PORT=8080 ./firesage
```

---

## Deploying to Google Cloud Run

FireSage includes a multi-stage, zero-dependency distroless root [Dockerfile](Dockerfile) exposing `PORT=8080` (`HOST=0.0.0.0`), matching the **StockSage** Cloud Run deployment workflow:

### Option A: Continuous Deployment via Google Cloud Console (Recommended)
1. Open **Google Cloud Console → Cloud Run → Create Service**.
2. Choose **Continuously deploy from a repository (GitHub)** and select `ausanyal/firesage` (`main` branch).
3. Set Build Type to **Dockerfile** (`/Dockerfile`).
4. Set Authentication to **Allow unauthenticated invocations**, Memory to `512 MiB` (`1` or `2` vCPU), Port `8080`, and click **Create**.

### Option B: One-Command `gcloud` CLI Deploy
```bash
gcloud run deploy firesage \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --cpu 2
```

---

## Architecture & Quantitative Specification

See [docs/ARCHITECTURE_AND_DESIGN.md](docs/ARCHITECTURE_AND_DESIGN.md) for the complete mathematical formulation, tax/healthcare rule tables, and systems architecture.
