discordgateway, as the name suggests, is for the gateway only. However, the voice websocket logic is fairly similar and we can quickly derive a voice implementation as well (except udp) using the base_state.

operation codes and close codes uses the same type as the gateway, but their values are stored in this sub pkg.