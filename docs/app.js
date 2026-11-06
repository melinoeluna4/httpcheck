const statusButtons = [...document.querySelectorAll("[data-status]")];
const statusDetail = document.querySelector(".status-detail");

for (const button of statusButtons) {
  button.addEventListener("click", () => {
    for (const other of statusButtons) {
      const active = other === button;
      other.classList.toggle("is-active", active);
      other.setAttribute("aria-pressed", String(active));
    }

    if (statusDetail) {
      statusDetail.textContent = button.getAttribute("data-detail") ?? "";
    }
  });
}

const copyButtons = document.querySelectorAll("[data-copy]");

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const value = button.getAttribute("data-copy");
    if (!value) return;

    const original = button.textContent;

    try {
      await navigator.clipboard.writeText(value);
      button.textContent = "Copiado";
    } catch {
      button.textContent = "Copia manualmente";
    }

    window.setTimeout(() => {
      button.textContent = original;
    }, 1600);
  });
}
