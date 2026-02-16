# Contributing to TermPlay

Hey! Thanks for checking out TermPlay. If you're looking to help out, you're awesome.

We're building a fun, SSH-based game platform, and we'd love your help to make it better. Whether it's fixing a bug, adding a new game, or just improving the docs, everything counts.

## How to Get Started

1.  **Fork the repo** and clone it locally.
2.  Check the `README.md` to get the local dev server running.
3.  Play around! If you see something broken or have a cool idea, open an Issue first so we can chat about it.

## Adding a New Game

We designed the code to be modular (check `internal/chess` or `internal/snake` for examples). If you want to add a new game:
1.  Create a new package in `internal/yourgame`.
2.  Implement the game logic and a View function.
3.  Hook it up in `internal/ui`.

## Submitting Changes

1.  Create a new branch for your feature: `git checkout -b my-cool-feature`.
2.  Make your changes.
3.  Test it! (Run `go test ./...` and actually play the game).
4.  Push to your fork and open a Pull Request.

Keep your PR description clear—tell us *what* you changed and *why*. Screenshots are a huge plus if you changed the UI.

## Code Style

We stick to standard Go conventions. Run `go fmt ./...` before submitting. Simple as that.

Thanks for hacking with us!
