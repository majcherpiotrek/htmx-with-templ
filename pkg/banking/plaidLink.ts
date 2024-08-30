import { getServerProps } from "../common/utils/utils";

const props = getServerProps("plaidToken", window.Zod.object({ token: window.Zod.string() }));

const PLAID_LINK_BUTTON_ID = "plaidLinkButton"

const handler = window.Plaid.create({
  token: props.token,
  onSuccess: (publicToken, meta) => {
    console.log("success");
    console.log({ publicToken, meta });
    window.htmx.ajax("POST", "http://localhost:42069/banks", {
      values: {
        publicToken
      },
      source: `#${PLAID_LINK_BUTTON_ID}`,
      swap: "none"
    });
  },
  onExit: (error, metadata) => {
    console.log("exit");
    console.log({ error, metadata });
  },
  onEvent: (eventName, metadata) => {
    console.log("event");
    console.log({ eventName, metadata });
  }
});

document.addEventListener("DOMContentLoaded", () => {
  const plaidLinkButton = document.getElementById(PLAID_LINK_BUTTON_ID);
  console.log("added on click listener to plaid button");

  plaidLinkButton?.addEventListener("click", () => { handler.open(); })
})
