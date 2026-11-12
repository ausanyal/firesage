# FireSage 🔥

**Institutional-grade Stochastic FIRE (Financial Independence, Retire Early) & Lifecycle Simulation Platform for India (`₹`) and the USA (`$`).**

Built with a **Go** concurrent Monte Carlo backend + **React / TypeScript** interactive visualization suite compiled into a **single static binary** for **Google Cloud Run**.

## Key Capabilities

- **Dual-Jurisdiction Native Engine (`🇮🇳 India` & `🇺🇸 USA`):**
  - **India (`₹ L / Cr`):** Sensex/Nifty TRI, RBI G-Sec & INR Gold series (1979+), Equity LTCG (`12.5%` > ₹1.25L), slab-rate debt taxation, Arbitrage bucket optimization, EPF/PPF/NPS lock-in & liquidity-gap detection, and `12%–14%` medical inflation shields.
  - **USA (`$K / $M`):** Shiller (1871+) & Damodaran (1928+) datasets, Federal + State tax engines, 5-year Roth Conversion Ladder, `72(t)` SEPP, **2026 ACA 400% FPL Subsidy Cliff** → Age 65 Medicare transition, and Social Security PIA bend-point optimization.
- **City-Tier Domestic Geo-Arbitrage:**
  - **India:** Compare retiring in Tier-1 metros (*Bengaluru, Mumbai, Delhi-NCR*) vs. Tier-2 / lifestyle cities (*Goa, Mysuru, Coimbatore, Kochi*).
  - **USA:** Compare retiring in Tier-1 VHCOL cities (*San Jose CA, NYC, Seattle*) vs. Tier-2 MCOL hubs (*Raleigh NC, Austin TX, Tampa FL*) including state income/capital-gains tax arbitrage.
- **4-Mode Goroutine Simulation Core:** Stationary Block Bootstrap (`10,000` paths in `<15ms`), Historical Crisis Replay, Valuation/CAPE-Conditioned Regime-Switching Monte Carlo, and Custom Black-Swan Stress Tests.
- **Local-First Privacy, Cloud-Extensible:** Zero-login `IndexedDB` persistence + JSON export/import out of the box, architected behind a pluggable `PlanRepository` interface for **Google Sign-In + Cloud Firestore** sync.

## Architecture & Blueprint

See [docs/ARCHITECTURE_AND_DESIGN.md](docs/ARCHITECTURE_AND_DESIGN.md) for the complete product, quantitative, and systems design specification.
