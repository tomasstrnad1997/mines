* Gameplay
    * First move safe (ensure the board is similar with same seed and different first move)
    * Alternative - first move automatically
    * Maybe more gamemodes???
* Storing game history in DB
* Logs
* Players
    * Names
* Game
    * Score
    * Time
* UI
    * Mine count
    * Player colors (colored square border)
    * Player scores
    * Show who lost the game...
    * Visually show which mine other player clicked
    * Pings/drawing for other players
* Matchmaking server
    * Spawn game server
        * Monitor
    * Log server creation
    * Server browser
    * Add option to tell the MM server that some game server is running
        * Add locally hosted games to browser (filter official games and unoficial)
        * If the matchmaking server resets it needs to sync with currently running game
    * Option to connect by game server name + password
* Game Server Launcher
    * For official games create a game lancher that spawns game servers
    * Can run on any number of machines
        * MM server either gets their locations from database or game server can connect to MM server and announce itself (unoficiall game server)
    * Layer between matchmaking server and game server
* Game Server
    * Server names
    * Public/private
    * Passwords
    * ?Whitelist friends?
