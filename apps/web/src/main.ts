import { createServer } from "./server.ts";

const port = Number(process.env.SAFE_WEB_PORT ?? process.env.PORT ?? "3000");
const server = createServer();

server.listen(port, () => {
  process.stdout.write(`safe web client listening on http://127.0.0.1:${port}\n`);
});
