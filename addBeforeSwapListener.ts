document.addEventListener("DOMContentLoaded", () => {
  console.log("Adding htmx:beforeSwap event listener ...");

  document.body.addEventListener("htmx:beforeSwap", function(evt) {
    if (evt.detail.xhr.status === 422 || evt.detail.xhr.status === 409) {
      evt.detail.shouldSwap = true;
      evt.detail.isError = false;
    }
  });
});
