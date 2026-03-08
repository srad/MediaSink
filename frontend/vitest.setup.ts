import { afterAll, afterEach, beforeAll } from "vitest";
import { setupServer } from "msw/node";
import { restHandlers } from "./tests/mocks/handlers";

export const server = setupServer(...restHandlers);

/*
server.events.on("request:start", ({ request }) => {
  console.log("Outgoing:", request.method, request.url);
});
 */

beforeAll(() =>
  server.listen({
    onUnhandledRequest: "error",
  }),
);
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

/*
const pinia = createTestingPinia();
setActivePinia(pinia);
*/
