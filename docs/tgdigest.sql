-- =============================================================================
-- Diagram Name: apisrv
-- Created on: 06.10.2025 20:40:43
-- Diagram Version: 
-- =============================================================================

CREATE TABLE "messageReactions" (
	"messageId" int8 NOT NULL,
	"chatId" int8 NOT NULL,
	"reactionsCount" int4 NOT NULL DEFAULT 0,
	"createdAt" timestamp with time zone NOT NULL DEFAULT now(),
	PRIMARY KEY("chatId","messageId")
);



