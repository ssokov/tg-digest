# Reactions bot

1. Copy local.toml.dist as local.toml in same directory, add your bot token
2. Init database using tgdigest.sql file, set coorect db credentials in local.toml
3. Use 'make run' command to run bot, use go 1.24+, or use default run option with flags '-config=cfg/local.toml -verbose -verbose-sql'
