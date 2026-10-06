# PayRam Clone - Mermaid Diagrams

**Purpose:** Machine-renderable diagrams for documentation and planning

---

## 1. System Context Diagram

```mermaid
graph TB
    subgraph Clients
        CU[Customer Browser]
        MD[Merchant Dashboard]
        MA[Mobile App]
        AI[AI Agent / MCP Client]
        WG[Embedded Widget]
    end

    subgraph "PayRam Platform (Self-Hosted VPS)"
        NX[Nginx Reverse Proxy + SSL]
        FE[Next.js Frontend :3000]
        BE[Go API Server :8080]
        MC[MCP Server :3333]
        WK[Background Workers]
        PG[(PostgreSQL :5432)]
        RD[(Redis :6379)]
        KS[Encrypted Key Store]
    end

    subgraph "Blockchain Networks"
        ETH[Ethereum]
        BTC[Bitcoin]
        BASE[Base]
        POLY[Polygon]
        TRON[Tron]
    end

    subgraph "Smart Contracts"
        SS[SmartSweep Contract]
        CW[Cold Wallet]
    end

    subgraph "External Services"
        OR[Onramp Provider]
        RPC[RPC Nodes - Alchemy/Infura]
    end

    CU --> NX
    MD --> NX
    MA --> NX
    AI --> MC
    WG --> NX

    NX --> FE
    NX --> BE
    NX --> MC

    BE --> PG
    BE --> RD
    BE --> KS
    MC --> BE
    WK --> PG
    WK --> RD

    WK --> RPC
    RPC --> ETH
    RPC --> BTC
    RPC --> BASE
    RPC --> POLY
    RPC --> TRON

    SS --> CW
    WK --> SS

    BE --> OR
```

---

## 2. Payment Flow

```mermaid
sequenceDiagram
    participant C as Customer
    participant W as Widget/Checkout
    participant API as Go API Server
    participant WS as Wallet Service
    participant BM as Block Monitor
    participant BC as Blockchain
    participant WH as Webhook Service
    participant M as Merchant Backend

    C->>W: Select amount & pay
    W->>API: POST /api/v1/payment
    API->>WS: Generate deposit address
    WS->>WS: HD derive (BIP-32/44)
    WS-->>API: Deposit address + QR
    API-->>W: {reference_id, address, url}
    W-->>C: Show QR code + address

    C->>BC: Send crypto payment

    loop Block Monitoring
        BM->>BC: Poll new blocks
        BC-->>BM: Block data
        BM->>BM: Match deposit addresses
    end

    BM->>API: Payment detected
    API->>API: Status = CONFIRMING

    loop Wait for confirmations
        BM->>BC: Check confirmations
        BC-->>BM: Confirmation count
    end

    BM->>API: Confirmations >= threshold
    API->>API: Status = CONFIRMED
    API->>WH: Queue webhook delivery
    WH->>M: POST webhook (HMAC signed)
    M-->>WH: 200 OK

    Note over W,C: Customer sees confirmation
```

---

## 3. SmartSweep Flow

```mermaid
sequenceDiagram
    participant SCH as Sweep Scheduler
    participant DB as PostgreSQL
    participant HW as Hot Wallet (Gas)
    participant SC as SmartSweep Contract
    participant DW as Deposit Wallet
    participant CW as Cold Wallet
    participant BC as Blockchain

    loop Sweep Interval
        SCH->>DB: Check deposit wallet balances
        DB-->>SCH: Wallets above threshold

        SCH->>BC: Estimate gas cost
        BC-->>SCH: Gas estimate

        SCH->>HW: Sign sweep transaction
        HW-->>SCH: Signed tx

        SCH->>SC: Execute sweep(token, amount)
        SC->>DW: Transfer funds
        DW->>CW: Funds → Cold Wallet (HARDCODED)
        BC-->>SCH: Tx confirmed

        SCH->>DB: Log sweep record
    end
```

---

## 4. Component Architecture

```mermaid
graph TB
    subgraph "Frontend (Next.js)"
        LP[Landing Page]
        DP[Demo Page]
        BG[Blog]
        DB[Dashboard]
        PM[Payments]
        WL[Wallets]
        SW[Sweeps]
        AN[Analytics]
        ST[Settings]
        LN[Login]
    end

    subgraph "API Layer (Go)"
        RT[HTTP Router - Gin]
        MW[Middleware Chain]
        PC[Payment Controller]
        WC[Wallet Controller]
        SC[Sweep Controller]
        AC[Analytics Controller]
        UC[Auth Controller]
        HC[Webhook Controller]
    end

    subgraph "Service Layer (Go)"
        PS[Payment Service]
        WLS[Wallet Service]
        SS[Sweep Service]
        WHS[Webhook Service]
        BCS[Blockchain Service]
        ANS[Analytics Service]
    end

    subgraph "Worker Layer (Go)"
        BME[Block Monitor - ETH]
        BMB[Block Monitor - BTC]
        BMT[Block Monitor - TRX]
        BMP[Block Monitor - Polygon]
        BMX[Block Monitor - Base]
        SWK[Sweep Worker]
        WHR[Webhook Retry Worker]
        EXP[Payment Expiry Worker]
    end

    subgraph "MCP Layer (TypeScript)"
        MPS[MCP Protocol Server]
        MTL[Tool Handlers]
        MIC[Internal API Client]
    end

    subgraph "Data Layer"
        PG[(PostgreSQL)]
        RD[(Redis)]
        EKS[Encrypted Key Store]
    end

    DB --> RT
    PM --> RT
    WL --> RT
    SW --> RT
    AN --> RT
    ST --> RT

    RT --> MW --> PC
    MW --> WC
    MW --> SC
    MW --> AC
    MW --> UC
    MW --> HC

    PC --> PS
    WC --> WLS
    SC --> SS
    HC --> WHS
    AC --> ANS

    PS --> PG
    PS --> RD
    WLS --> EKS
    SS --> BCS
    WHS --> RD

    BME --> BCS
    BMB --> BCS
    BMT --> BCS
    BMP --> BCS
    BMX --> BCS
    BCS --> PG

    MPS --> MTL --> MIC --> RT
```

---

## 5. Database ERD

```mermaid
erDiagram
    MERCHANTS {
        uuid id PK
        string name
        string email
        string password_hash
        json cold_wallets
        json settings
        timestamp created_at
    }

    API_KEYS {
        uuid id PK
        uuid merchant_id FK
        string key_hash
        string name
        string[] permissions
        boolean active
        timestamp last_used_at
        timestamp created_at
    }

    PAYMENTS {
        uuid id PK
        uuid merchant_id FK
        string reference_id UK
        decimal amount
        string currency
        string blockchain
        string status
        string deposit_address
        string txid
        integer confirmations
        string customer_email
        string customer_id
        string invoice_id
        timestamp expires_at
        timestamp confirmed_at
        timestamp created_at
    }

    WALLETS {
        uuid id PK
        uuid merchant_id FK
        string blockchain
        string address
        string type
        decimal balance
        string token
        integer derivation_idx
        timestamp created_at
    }

    SWEEPS {
        uuid id PK
        uuid from_wallet_id FK
        string to_address
        decimal amount
        string token
        string blockchain
        string txid
        string status
        decimal gas_used
        timestamp completed_at
        timestamp created_at
    }

    WEBHOOKS {
        uuid id PK
        uuid merchant_id FK
        string url
        string[] events
        string secret
        boolean active
        timestamp created_at
    }

    WEBHOOK_DELIVERIES {
        uuid id PK
        uuid webhook_id FK
        uuid payment_id FK
        string event
        json payload
        string status
        integer response_code
        integer attempts
        timestamp next_retry_at
        timestamp delivered_at
    }

    PAYOUTS {
        uuid id PK
        uuid merchant_id FK
        string blockchain_code
        string currency_code
        decimal amount
        decimal amount_in_usd
        string to_address
        string recipient_email
        string status
        string txid
        timestamp created_at
    }

    ACTIVITY_LOGS {
        uuid id PK
        uuid merchant_id FK
        string action
        string entity_type
        string entity_id
        json details
        string ip_address
        timestamp created_at
    }

    MERCHANTS ||--o{ API_KEYS : has
    MERCHANTS ||--o{ PAYMENTS : processes
    MERCHANTS ||--o{ WALLETS : owns
    MERCHANTS ||--o{ WEBHOOKS : configures
    MERCHANTS ||--o{ PAYOUTS : initiates
    MERCHANTS ||--o{ ACTIVITY_LOGS : generates
    WALLETS ||--o{ SWEEPS : sweeps_from
    WEBHOOKS ||--o{ WEBHOOK_DELIVERIES : delivers
    PAYMENTS ||--o{ WEBHOOK_DELIVERIES : triggers
```

---

## 6. Deployment Architecture

```mermaid
graph TB
    subgraph "Internet"
        DNS[DNS - your-domain.com]
    end

    subgraph "VPS - Ubuntu 22.04+"
        subgraph "Docker Compose"
            NGX[nginx:latest<br/>Port 80/443<br/>SSL + Reverse Proxy]
            API[payram-api<br/>Go Binary<br/>Port 8080/8443]
            WEB[payram-web<br/>Next.js<br/>Port 3000]
            MCP[payram-mcp<br/>Node.js<br/>Port 3333]
            WRK[payram-workers<br/>Go Binary<br/>Background Jobs]
            PG[(postgres:16<br/>Port 5432)]
            RED[(redis:7<br/>Port 6379)]
        end

        subgraph "Volumes"
            VPG[pg_data]
            VRD[redis_data]
            VCR[certs/]
            VKY[keys/ encrypted]
            VLG[logs/]
        end

        CRT[Certbot<br/>Let's Encrypt]
    end

    DNS --> NGX
    NGX --> API
    NGX --> WEB
    NGX --> MCP
    API --> PG
    API --> RED
    WRK --> PG
    WRK --> RED
    PG --- VPG
    RED --- VRD
    NGX --- VCR
    API --- VKY
    WRK --- VLG
    CRT --> VCR
```

---

## 7. Security Layers

```mermaid
graph TB
    subgraph "Layer 1: Network"
        FW[UFW Firewall<br/>Only 80, 443]
        F2B[fail2ban<br/>Brute Force Protection]
        SSL[TLS 1.3<br/>Let's Encrypt]
    end

    subgraph "Layer 2: Application"
        AK[API Key Auth<br/>SHA-256 Hashed]
        RL[Rate Limiting<br/>Redis-based]
        IV[Input Validation<br/>Strict Types]
        CORS[CORS Policy<br/>Strict Origins]
        CSP[CSP Headers]
    end

    subgraph "Layer 3: Webhook Security"
        HMAC[HMAC-SHA256<br/>Signatures]
        TS[Timestamp<br/>Validation]
        IDEM[Idempotency<br/>Keys]
    end

    subgraph "Layer 4: Key Management"
        ZKE[Zero Key Exposure<br/>No deposit keys on server]
        AES[AES-256<br/>Hot wallet only]
        HSC[Hardcoded<br/>Smart Contract Destinations]
        OFF[Offline<br/>Master Keys]
    end

    subgraph "Layer 5: Data"
        TDE[DB Encryption<br/>at Rest]
        BAK[Encrypted<br/>Off-site Backups]
        AUD[Complete<br/>Audit Trail]
    end

    FW --> AK
    AK --> HMAC
    HMAC --> ZKE
    ZKE --> TDE
```

---

## 8. User Role Hierarchy

```mermaid
graph TB
    OW[Owner<br/>Full Access - Cannot be removed]
    AD[Admin<br/>Manage all projects, users, payments]
    PL[Project Lead<br/>Create/update projects, analytics]
    PM[Project Manager<br/>View-only, export reports]
    PO[Project Ops<br/>View payment/customer data]
    PRA[Platform Referral Admin<br/>Full referral program access]

    OW --> AD
    AD --> PL
    PL --> PM
    PM --> PO
    AD --> PRA
```

---

## 9. Payment Status State Machine

```mermaid
stateDiagram-v2
    [*] --> OPEN: Payment Created
    OPEN --> CONFIRMING: Transaction Detected
    OPEN --> CANCELLED: Expiration Timer
    CONFIRMING --> FILLED: Full Amount + Confirmations
    CONFIRMING --> PARTIALLY_FILLED: Underpayment
    CONFIRMING --> OVER_FILLED: Overpayment
    FILLED --> SWEPT: SmartSweep Executed
    PARTIALLY_FILLED --> FILLED: Additional Payment
    SWEPT --> [*]
    CANCELLED --> [*]
```

---

## 10. Payout Status State Machine

```mermaid
stateDiagram-v2
    [*] --> pending_otp: Payout Created
    pending_otp --> pending_approval: OTP Verified
    pending_approval --> pending: Admin Approved
    pending_approval --> rejected: Admin Declined
    pending --> initiated: Broadcast to Chain
    initiated --> sent: Tx Confirmed
    initiated --> failed: Tx Error
    sent --> processed: On-chain Final
    pending_otp --> cancelled: User Cancelled
    pending_approval --> cancelled: Admin Cancelled
    processed --> [*]
    failed --> [*]
    rejected --> [*]
    cancelled --> [*]
```
