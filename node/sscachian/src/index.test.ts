import assert from "node:assert/strict";
import test from "node:test";

import { SSCACHIAN_RUNTIME, placeholder } from "./index.js";

test("placeholder exports runtime id", () => {
  assert.equal(SSCACHIAN_RUNTIME, "node");
  assert.match(placeholder(), /not yet implemented/);
});
