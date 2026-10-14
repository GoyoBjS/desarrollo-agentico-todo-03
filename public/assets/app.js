// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  var pendingCount = document.querySelector("[data-pending-count]");
  var pendingLabel = document.querySelector("[data-pending-label]");
  if (pendingCount && pendingLabel) {
    var pending = document.querySelectorAll(".task:not(.task-completed)").length;
    pendingCount.textContent = String(pending);
    pendingLabel.textContent = pending === 1 ? "pendiente" : "pendientes";
  }

  var clearCompleted = document.querySelector("[data-clear-completed]");
  if (clearCompleted) {
    clearCompleted.hidden = document.querySelectorAll(".task.task-completed").length === 0;
  }

  fetch(form.action);
});
