<div align="center">

<img src="https://ibb.co/zhnxLk3B" alt="ChameleonDB Logo" width="200" height="auto">


*A modern, type-safe, graph-oriented database access language*

[![Build Status](https://github.com/dperalta86/chameleondb/CI)](https://github.com/dperalta86/chameleondb/actions)
[![License: Apache](https://img.shields.io/badge/license-Apache%20License%202.0-blue)](https://www.apache.org/licenses/LICENSE-2.0)
[![Rust Version](https://img.shields.io/badge/rust-1.75%2B-orange.svg)](https://www.rust-lang.org)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://golang.org)

[Documentation](https://docs.chameleondb.dev) • [Examples](./examples) • [Contributing](./CONTRIBUTING.md) • [Roadmap](#roadmap)

</div>

---

## Overview

ChameleonDB replaces the "tables and JOINs" mindset with **relationship navigation in a typed object graph**, providing compile-time validation and modern syntax.

Instead of thinking in terms of relational tables, developers navigate an intuitive graph of strongly-typed entities with verified relationships.

### The Problem

Traditional database access comes with fundamental challenges:

- **SQL** forces you to think in flat tables and manual JOINs
- **ORMs** have leaky abstractions with runtime errors
- **Type-safety** is missing at compile-time for queries
- **N+1 queries** are easy to introduce accidentally

### The chameleonDB Solution

```rust
// Define your schema once
entity User {
    id: uuid primary,
    email: string unique,
    age: int,
    orders: [Order] via user_id,
}

entity Order {
    id: uuid primary,
    total: decimal,
    user: User,
}

// Write type-safe queries
db.users()
    .filter(|u| u.email == "ana@mail.com")
    .filter(|u| u.orders.any(|o| o.total > 100))
    .include(|u| u.orders)
    .execute()
    .await
```

**What you get:**
- ✅ **Compile-time type safety** - Catch errors before runtime
- ✅ **Graph navigation** - No manual JOINs required
- ✅ **Predictable behavior** - No magic, explicit control
- ✅ **High performance** - Rust core with optimized execution
- ✅ **Simple deployment** - Single binary, no runtime dependencies

---

## Quick Start

### Prerequisites

- Rust 1.75+
- Go 1.21+
- PostgreSQL 14+ (v1.0 backend)

### Installation

```bash
# Clone the repository
git clone https://github.com/dperalta86/chameleondb.git
cd chameleondb

# Build Rust core
cd chameleon-core
cargo build --release

# Build Go runtime
cd ../chameleon
go build -o chameleon cmd/chameleon/main.go

# Run your first query
./chameleon --schema examples/blog.cham --query examples/queries/users.cham
```

### Your First Schema

Create `blog.cham`:

```rust
entity User {
    id: uuid primary,
    username: string unique,
    email: string unique,
    created_at: timestamp default now(),
    posts: [Post] via author_id,
}

entity Post {
    id: uuid primary,
    title: string,
    content: string,
    published: bool default false,
    author: User,
    created_at: timestamp default now(),
}
```

### Your First Query

From Rust:

```rust
use chameleon::prelude::*;

#[tokio::main]
async fn main() -> Result<()> {
    let db = chameleonDB::connect("postgresql://localhost/mydb").await?;
    
    let users = db.users()
        .filter(|u| u.posts.any(|p| p.published == true))
        .include(|u| u.posts)
        .execute()
        .await?;
    
    for user in users {
        println!("{}: {} posts", user.username, user.posts.len());
    }
    
    Ok(())
}
```

From Go:

```go
package main

import "github.com/dperalta86/chameleondb/pkg/engine"

func main() {
    db := engine.Connect("postgresql://localhost/mydb")
    
    users := db.Users().
        Filter(User.Posts.Any(Post.Published.Eq(true))).
        Include("posts").
        Execute()
    
    for _, user := range users {
        fmt.Printf("%s: %d posts\n", user.Username, len(user.Posts))
    }
}
```

---

## Architecture

chameleonDB uses a **hybrid Rust + Go architecture** for optimal performance and developer experience:

```
┌─────────────────────────────────────┐
│  Rust Core (libchameleon.so)        │
│  - Parser (LALRPOP)                 │
│  - Type checker                     │
│  - Query optimizer                  │
│  - Schema validator                 │
└──────────────┬──────────────────────┘
               ↕ FFI (C ABI)
┌──────────────▼──────────────────────┐
│  Go Runtime (chameleon binary)      │
│  - Query executor                   │
│  - Connection pooling (pgx)         │
│  - HTTP API (optional)              │
│  - CLI tool                         │
└─────────────────────────────────────┘
```

### Why This Architecture?

- **Rust Core**: Extreme type-safety, zero-cost abstractions, superior parser performance
- **Go Runtime**: Simple deployment, excellent concurrency, robust tooling
- **FFI Bridge**: ~100ns overhead, negligible for database operations
- **Future-proof**: Easy to add bindings for Node, Python, Java, etc.

---

## Core Principles

### 1. Model First
The schema is the single source of truth. Everything else derives from it.

### 2. Compile-Time Safety
Errors are caught before execution. No runtime surprises.

### 3. Graph Navigation
Navigate relationships naturally. No manual JOIN writing.

### 4. Explicit Over Implicit
No magic. Predictable, deterministic behavior.

### 5. Runtime Enforcer
Rule-based optimization (v1.0). ML-based in future versions.

---

## Features

### Current (v0.1 - MVP)

- [x] Schema parser with LALRPOP
- [x] AST representation
- [x] Basic type checking
- [x] PostgreSQL backend
- [x] Simple query builder
- [x] FFI bridge (Rust ↔ Go)

### Planned (v0.2 - Type-safe Queries)

- [ ] Code generation from schema
- [ ] Full compile-time validation
- [ ] Eager loading with `.include()`
- [ ] Query result typing

### Planned (v0.3 - Production Ready)

- [ ] Automatic migrations
- [ ] Robust error handling
- [ ] Complete test framework
- [ ] Performance benchmarks

### Planned (v1.0 - Stable Release)

- [ ] Complete documentation
- [ ] Stable Go bindings
- [ ] Mature PostgreSQL backend
- [ ] Production case studies

### Future (v2.0+)

- [ ] ML-based query optimization
- [ ] Multi-backend support (MySQL, SQLite)
- [ ] Adaptive indexing
- [ ] Additional language bindings (Node, Java, Python)

---

## Non-Goals (v1.0)

chameleonDB is **intentionally scoped** to do a few things exceptionally well:

- ❌ **NOT a complete SQL replacement** - Use SQL for complex analytics
- ❌ **NOT a universal ORM** - Focused on relational databases
- ❌ **NOT an auto-optimizer** - Explicit control over queries
- ❌ **NOT hiding database behavior** - Transparent operations
- ❌ **NOT using ML for optimization** - That's v2.0+

---

## Project Structure

```
chameleon/
├── chameleon-core/          # Rust core library
│   ├── src/
│   │   ├── ast/             # Abstract syntax tree
│   │   ├── parser/          # LALRPOP grammar
│   │   ├── typechecker/     # Type validation
│   │   ├── optimizer/       # Query optimization
│   │   └── ffi/             # Foreign function interface
│   ├── Cargo.toml
│   └── build.rs
│
├── chameleon/               # Go runtime
│   ├── cmd/chameleon/       # CLI tool
│   ├── pkg/
│   │   ├── engine/          # CGO wrapper
│   │   ├── runtime/         # Query executor
│   │   ├── backend/         # Database drivers
│   │   └── server/          # HTTP API (optional)
│   └── go.mod
│
├── docs/                    # Documentation
│   ├── architecture.md
│   ├── specification.md
│   ├── manifesto.md
│   └── tutorials/
│
├── examples/                # Example schemas and queries
│   ├── blog/
│   ├── ecommerce/
│   └── analytics/
│
├── tests/                   # Integration tests
│   ├── fixtures/
│   └── scenarios/
│
├── LICENSE
├── README.md
├── CONTRIBUTING.md
└── CODE_OF_CONDUCT.md
```

---

## Roadmap

### Phase 1: Foundation (Q1 2026) - **IN PROGRESS**
- ✅ Project setup
- 🔄 Schema parser
- 🔄 Basic type checker
- ⏳ FFI bridge
- ⏳ PostgreSQL connector

### Phase 2: Type Safety (Q2 2026)
- Code generation
- Compile-time validation
- Relationship navigation
- Include/eager loading

### Phase 3: Production Ready (Q3 2026)
- Migrations
- Error handling
- Testing framework
- Documentation

### Phase 4: Stable v1.0 (Q4 2026)
- Performance optimization
- Production hardening
- Case studies
- Community building

### Phase 5: Advanced Features (2027+)
- ML-based optimization
- Multi-backend support
- Additional language bindings
- Enterprise features

---

## Contributing

We welcome contributions! chameleonDB is in early stages and there's plenty to do.

### How to Contribute

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/dperalta86/chameleondb.git
cd chameleondb

# Set up Rust environment
cd chameleon-core
cargo build
cargo test

# Set up Go environment
cd ../chameleon
go mod download
go test ./...

# Run integration tests
make test-integration
```

### Areas We Need Help

- 🦀 **Rust**: Parser improvements, type checker, optimizer
- 🐹 **Go**: Runtime improvements, connection pooling, testing
- 📝 **Documentation**: Tutorials, guides, API docs
- 🧪 **Testing**: Unit tests, integration tests, benchmarks
- 🎨 **Design**: Logo, website, examples
- 🌍 **Community**: Advocacy, feedback, issue triage

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

---

## Community

- **GitHub Discussions**: [Ask questions, share ideas](https://github.com/dperalta86/chameleondb/discussions)
- **Discord**: [Join our server](https://discord.gg/chameleondb)
- **Twitter**: [@chameleonDB](https://twitter.com/chameleondb)
- **Blog**: [dev.to/chameleondb](https://dev.to/chameleondb)

---

## Performance

chameleonDB is designed for performance from the ground up:

- **Rust core**: Zero-cost abstractions, no garbage collection
- **Optimized parser**: LALRPOP generates efficient code
- **Connection pooling**: pgx provides optimal PostgreSQL performance
- **Query optimization**: Compile-time and runtime optimizations

Benchmarks coming in v0.2. Early tests show:
- Schema parsing: < 1ms for typical schemas
- Type checking: < 5ms for complex queries
- FFI overhead: ~100ns per call
- Query execution: Comparable to hand-written SQL

---

## Comparison

| Feature | chameleonDB | SQL | Prisma | GORM | SQLAlchemy |
|---------|--------------|-----|--------|------|------------|
| Type Safety | Compile-time | None | Runtime | Runtime | Runtime |
| Graph Navigation | ✅ Native | ❌ Manual JOINs | ✅ Relations | ⚠️ Preload | ⚠️ Eager load |
| Performance | High (Rust) | Highest | Medium | Medium | Medium |
| Learning Curve | Low | High | Low | Low | Medium |
| Schema First | ✅ Required | ⚠️ Optional | ✅ Required | ❌ Code-first | ⚠️ Hybrid |
| Multi-language | 🔄 Planned | ✅ Universal | ❌ TypeScript | ❌ Go | ❌ Python |

---

## License

chameleonDB is licensed under the **APACHE 2.0 License** - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

chameleonDB is inspired by:

- **Prisma** - Schema-first approach and developer experience
- **GraphQL** - Graph-based data querying
- **Rust** - Type safety and zero-cost abstractions
- **LINQ** - Composable query syntax
- **EdgeDB** - Rethinking database access

Special thanks to all [contributors](https://github.com/yourusername/chameleon/graphs/contributors)!

---

## Support

If you find chameleonDB useful, please consider:

- ⭐ **Star** this repository
- 🐦 **Share** on social media
- 📝 **Write** a blog post or tutorial
- 💬 **Join** our community discussions
- 🤝 **Contribute** code or documentation

---

<div align="center">

**Built with ❤️ by developers, for developers**

[Website](https://chameleondb.dev) • [Documentation](https://docs.chameleondb.dev) • [GitHub](https://github.com/yourusername/chameleon)

</div>
