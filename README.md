# botsrv - quick startup of new telegram bots

## Architecture and packages:
- cfg files: create cfg/local.toml file, use local.toml.dist as example
- docs: use MicroOLAP DB Designer to edit botsrv.pdd, generate botsrv.sql from it
- Makefile: use created instructions to quick codegen & bot running
- db: use vmkteam/mfd library to generate ORM structures
- pkg/botsrv: all business-logic
- pkg/app: building starting architecture and starting bot & servers
- pkg/rpc: jsonrpc server handlers
