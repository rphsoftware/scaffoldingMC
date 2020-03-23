# Scaffolding MC

A minecraft server with the main purpose being previewing map files. VERY EARLY ALPHA STAGE!

What currently works:

- Reading region files (only with both x and y positive)
- Sending rudimentary chunk data
- Login in both online and offline mode
- Multiple clients at the same time
- Keepalive

What needs to be done:

- Clients seeing each other
- Getting player position and updating what chunks get sent
- Sending empty chunks at the edges
- Negative positions for region files that actually work (currently the chunks are in a wrong order)
- Streaming of chunk data rather than pregeneration (That was a lazy hack)
- Significant reduction of memory usage (this goes back to the previous point)
