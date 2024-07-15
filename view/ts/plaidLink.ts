const props = JSON.parse(document.getElementById("plaidToken")?.textContent ?? "");

console.log("read props added by server", props);

const handler = window.Plaid.create({
  token: props["token"],
  onSuccess: (publicToken, meta) => {
    console.log("success");
    console.log({ publicToken, meta });
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

console.log("handler", handler);


document.addEventListener("DOMContentLoaded", () => {
  const plaidLinkButton = document.getElementById("plaidLinkButton");
  console.log("added on click listener to plaid button");

  plaidLinkButton?.addEventListener("click", () => { handler.open(); })
})
