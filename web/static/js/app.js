(function () {
  "use strict";

  function openDialog(id) {
    var dialog = document.getElementById(id);
    if (dialog && typeof dialog.showModal === "function") {
      dialog.showModal();
    }
  }

  function closeDialog(element) {
    var dialog = element.closest("dialog");
    if (dialog) {
      dialog.close();
    }
  }

  function markTodo(card, isDone) {
    var checkbox = card.querySelector(".checkbox");
    if (checkbox) {
      checkbox.dataset.done = isDone ? "true" : "false";
      checkbox.textContent = isDone ? "✓" : "";
      checkbox.setAttribute("aria-checked", isDone ? "true" : "false");
    }
    card.classList.toggle("card-done", isDone);
  }

  function toggleTodo(button) {
    var card = button.closest("[data-todo-card]");
    var todoID = button.dataset.todoId;
    if (!card || !todoID) {
      return;
    }

    button.disabled = true;

    fetch("/todos/" + encodeURIComponent(todoID) + "/toggle", {
      method: "POST",
      headers: { Accept: "application/json" }
    })
      .then(function (response) {
        if (!response.ok) {
          throw new Error("сервер вернул статус " + response.status);
        }
        return response.json();
      })
      .then(function (payload) {
        markTodo(card, Boolean(payload.is_done));
        var filter = card.dataset.filter || "all";
        if (filter !== "all") {
          window.location.reload();
        }
      })
      .catch(function () {
        window.location.reload();
      })
      .finally(function () {
        button.disabled = false;
      });
  }

  document.addEventListener("click", function (event) {
    var opener = event.target.closest("[data-dialog]");
    if (opener) {
      event.preventDefault();
      openDialog(opener.dataset.dialog);
      return;
    }

    var closer = event.target.closest("[data-close]");
    if (closer) {
      event.preventDefault();
      closeDialog(closer);
      return;
    }

    var checkbox = event.target.closest(".checkbox[data-todo-id]");
    if (checkbox) {
      event.preventDefault();
      toggleTodo(checkbox);
    }
  });

  document.addEventListener("submit", function (event) {
    var form = event.target;
    var question = form.dataset.confirm;
    if (question && !window.confirm(question)) {
      event.preventDefault();
    }
  });

  document.addEventListener("keydown", function (event) {
    if (event.key !== "/" || event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }

    var active = document.activeElement;
    var tag = active ? active.tagName : "";
    if (tag === "INPUT" || tag === "TEXTAREA" || (active && active.isContentEditable)) {
      return;
    }

    var search = document.querySelector('header input[name="q"]');
    if (search) {
      event.preventDefault();
      search.focus();
    }
  });
})();
