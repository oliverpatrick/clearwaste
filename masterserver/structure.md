For what you’re building, I’d strongly favour a modular monolith in one Go repository, deployed as a few separate processes where isolation actually matters.

The important distinction is:

code architecture != deployment architecture

You can have:

one repository
one set of domain packages
one database schema
shared Go types/interfaces

while running:

account server
world 1
world 2
world 3
...

as separate processes.

That is probably the sweet spot for your MMO.

What I'd build initially

Your repo stays roughly:

mmo/
├── cmd/
│   ├── account/
│   └── server/
│
└── internal/
    ├── app/
    │   ├── accountserver/
    │   └── gameserver/
    │
    ├── engine/
    │   ├── ecs/
    │   ├── simulation/
    │   ├── spatial/
    │   └── network/
    │
    ├── account/
    ├── character/
    ├── world/
    ├── game/
    ├── social/
    ├── leaderboard/
    ├── content/
    └── storage/

But deployment might look like:

                        ┌──────────────┐
                        │ PostgreSQL   │
                        └──────┬───────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
        ┌───────▼────────┐                   │
        │ Account Server │                   │
        │                │                   │
        │ auth           │                   │
        │ characters     │                   │
        │ world list     │                   │
        └───────┬────────┘                   │
                │                            │
                │ tickets                    │
                │                            │
         ┌──────┴────────────────────────────┐
         │                                   │
         ▼                                   ▼
┌─────────────────┐                 ┌─────────────────┐
│     World 1     │                 │     World 2     │
│                 │                 │                 │
│ max 2000        │                 │ max 2000        │
│ ECS             │                 │ ECS             │
│ simulation      │                 │ simulation      │
│ spatial         │                 │ spatial         │
│ gameplay        │                 │ gameplay        │
└─────────────────┘                 └─────────────────┘

Both world processes are built from:

cmd/server

Just different configuration:

WORLD_ID=1 ./server
WORLD_ID=2 ./server
WORLD_ID=3 ./server

That's already horizontal scaling.

If you have:

World 1   2000
World 2   2000
World 3   2000
World 4   2000

you can support 8,000 concurrent players without one simulation needing to run 8,000 players.

Add worlds and you scale out.

And this isn't really a traditional monolith anymore

There's a useful distinction between three architectures.

1. Single-process monolith

    one process
        account
        game
        social
        everything

I wouldn't do that for your MMO.

Then:

2. Modular monolith + multiple runtime processes

    one codebase

    account process

    world 1 process
    world 2 process
    world 3 process

    shared domain packages
    shared storage

This is what I'd choose.

Then there's:

3. Microservices

    auth-service
    character-service
    world-directory-service
    presence-service
    friends-service
    clan-service
    leaderboard-service
    game-world-service
    save-service
    ...

I absolutely would not start there.

You'd create a huge amount of infrastructure before your MMO actually needs it.

Your existing boundaries already prepare you for extraction

This is why we've been separating:

account/
character/
world/
social/
leaderboard/
game/

rather than dumping everything into:

server/

You can initially call these packages directly:

characters, err := characterService.ListForAccount(
    ctx,
    accountID,
)

No network.

No JSON.

No gRPC.

No service discovery.

Just a Go function call.

Later, if character genuinely needs to become another service:

Before

Account Server
     │
     │ Go call
     ▼
character.Service

becomes:

After

Account Server
     │
     │ RPC
     ▼
Character Service
     │
     ▼
character.Service

The actual domain logic doesn't necessarily change much.

That's why clean module boundaries matter more right now than microservices.

Your world server is already the natural scaling unit

This is especially true given your 2,000 players per world model.

The world is basically a built-in shard.

You don't need to distribute one ECS simulation across ten machines.

Instead:

machine A
    World 1
    World 2

machine B
    World 3
    World 4

machine C
    World 5

Each has:

its own ECS
its own tick loop
its own spatial index
its own NPCs
its own players

That's vastly simpler than trying to horizontally distribute a single simulation.

And if World 4 dies:

World 1 ✓
World 2 ✓
World 3 ✓
World 4 ✗
World 5 ✓

You don't take the entire MMO down.

That's one of the reasons I'd keep each world as its own OS process even if you're running multiple worlds on the same physical machine.

The social stuff doesn't need to be microservices either

Initially your world server can have:

type Services struct {
    Characters  *character.Service
    Friends     *friends.Service
    Clans       *clans.Service
    Presence    *presence.Service
    Leaderboard *leaderboard.Service
}

and they can talk to shared persistence.

Conceptually:

World 1 ─┐
World 2 ─┼──── PostgreSQL
World 3 ─┘

Possibly with Redis eventually for things like:

sessions
online character locks
presence
short-lived world tickets
cross-world pub/sub

You don't need a dedicated presence microservice simply because multiple servers need presence information.

For instance:

World 1
   │
   └── Redis
        character:1001:world = 1

World 2
   │
   └── Redis
        character:8002:world = 2

Now all worlds can answer:

Where is character 1001?

without another application server.

Cross-world messaging is where it starts getting interesting

Suppose:

foo      → World 1
Alice    → World 3

and foo sends Alice a private message.

You don't want:

World 1 → directly open TCP connection → World 3

I'd eventually introduce a messaging mechanism:

                  Redis / NATS / similar
                         │
          ┌──────────────┼──────────────┐
          │              │              │
       World 1        World 2        World 3

World 1 publishes:

private-message:
    from: foo
    to: Alice
    message: hello

World 3 receives it.

That still doesn't mean you need microservices.

It's just communication between your world processes.

.sav files need one consideration

Your character .sav architecture is fine while everything runs on one machine.

But imagine:

Machine A
    World 1

Machine B
    World 2

and foo logs out from World 1 and later enters World 2.

You don't want:

Machine A:
/saves/1001/930.sav

Machine B:
    ???


So once worlds run across multiple physical machines, I'd change the save repository implementation, not your character architecture.

You already have:

character/save/
    repository.go

So maybe today:

type Repository interface {
    Load(ctx context.Context, id CharacterID) (*Save, error)
    Store(ctx context.Context, id CharacterID, save *Save) error
}

implementation:

storage/saves/local/

Later:

storage/saves/objectstore/

using an S3-compatible object store or another shared durable storage mechanism.

Your game code doesn't care.

That's another reason the repository abstraction is useful.

When would I actually split something into a microservice?

I wouldn't do it based on code size.

I would do it when deployment behaviour requires independence.

For example, imagine leaderboards become extremely busy:

world servers
      │
      │ millions of score updates
      ▼
leaderboard processing
      │
      ▼
ranking cache

And leaderboard updates start affecting the database latency needed for player saves.

That's a good reason to extract it:

World
  │
  │ ScoreChanged event
  ▼
message bus
  │
  ▼
Leaderboard Service

Or presence grows into:

100 worlds
200,000 online players
friends
cross-world messaging
clans
chat
mobile companion app
web presence

Now:

social/

might reasonably become:

cmd/social/

But notice the progression:

internal/social/

       ↓

same package, separate process

       ↓

potentially separately deployed service

You don't need to design the whole MMO around RPC upfront.

A likely long-term evolution

I could see your architecture naturally growing like this:

PHASE 1

Account
World 1
World 2

PostgreSQL
Redis

Then:

PHASE 2

                    Account
                       │
                       │
                   PostgreSQL
                       │
       ┌───────────────┼───────────────┐
       │               │               │
    World 1         World 2         World 3
       │               │               │
       └───────────┬───┴───────────────┘
                   │
                 Redis
           sessions/presence

Then perhaps eventually:

                        Gateway
                           │
                   ┌───────┴───────┐
                   │               │
                Account         Worlds
                   │               │
                   │       ┌───────┼────────┐
                   │       ▼       ▼        ▼
                   │      W1      W2       W3
                   │
        ┌──────────┼───────────────┐
        ▼          ▼               ▼
      Social   Leaderboards    Character Saves

But only extract the bottom row if there's a measurable reason.

One rule I'd establish now

Don't let modules reach directly into each other's database tables everywhere.

This:

// game/combat

db.Query(`
    SELECT member
    FROM accounts
    WHERE id = ?
`)

would be a bad direction.

Instead:

membership.IsActive(accountID)

Similarly:

characters.Load(characterID)
friends.List(characterID)
presence.SetOnline(characterID, worldID)

That keeps the logical boundaries intact while everything is still in-process.

If you eventually split something out, you're replacing the implementation behind a boundary instead of hunting down 300 raw SQL calls.

So for your project

I would aim for this:

                    REPOSITORY
                        │
      ┌─────────────────┼─────────────────┐
      │                 │                 │
 account code      game code        social code
 character code    engine code      world code
      │                 │                 │
      └─────────────────┼─────────────────┘
                        │
                    shared modules

but deploy:

                   ACCOUNT PROCESS
                         │
                         │
                 PostgreSQL / Redis
                         │
              ┌──────────┼──────────┐
              │          │          │
           WORLD 1    WORLD 2    WORLD 3

That's a modular monolith / monorepo with process-level sharding by world.

For an MMO with a hard 2,000-player-per-world model, I think that's a particularly strong architecture. Your world boundary already gives you one of the biggest benefits people try to get from microservices—horizontal scalability and fault isolation—without forcing every friends-list lookup or character load to become a network request.

I'd stay with that until you can point at a specific subsystem and say: “this needs to scale, deploy, fail, or be maintained independently from the rest.” That's the point where I'd consider extracting that particular module, rather than converting the whole system to microservices.