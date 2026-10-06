(function () {
  const script = document.currentScript as HTMLScriptElement;
  if (!script) return;

  const config = {
    url: script.getAttribute("data-payminto-url") || "",
    apiKey: script.getAttribute("data-api-key") || "",
    amounts: (script.getAttribute("data-amounts") || "10,50,100,200").split(",").map(Number),
    theme: script.getAttribute("data-theme") || "dark",
    customerEmail: script.getAttribute("data-customer-email") || "",
  };

  function createButton(): HTMLButtonElement {
    const btn = document.createElement("button");
    btn.textContent = "Pay with Payminto";
    btn.style.cssText = `
      padding: 12px 24px;
      background: #01e46f;
      color: #000;
      border: none;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      font-family: system-ui, sans-serif;
    `;
    btn.addEventListener("click", openOverlay);
    return btn;
  }

  function openOverlay(): void {
    const overlay = document.createElement("div");
    overlay.style.cssText = `
      position: fixed; top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(0,0,0,0.7); display: flex; align-items: center;
      justify-content: center; z-index: 99999;
    `;

    const modal = document.createElement("div");
    modal.style.cssText = `
      background: ${config.theme === "dark" ? "#1a1a2e" : "#fff"};
      color: ${config.theme === "dark" ? "#fff" : "#000"};
      border-radius: 16px; padding: 32px; max-width: 400px; width: 90%;
      font-family: system-ui, sans-serif;
    `;

    const title = document.createElement("h2");
    title.textContent = "Select Amount";
    title.style.cssText = "margin: 0 0 16px; font-size: 20px;";
    modal.appendChild(title);

    const grid = document.createElement("div");
    grid.style.cssText = "display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px;";

    config.amounts.forEach((amount) => {
      const btn = document.createElement("button");
      btn.textContent = `$${amount}`;
      btn.style.cssText = `
        padding: 12px; border: 1px solid #333; border-radius: 8px;
        background: transparent; color: inherit; font-size: 16px;
        cursor: pointer; font-weight: 600;
      `;
      btn.addEventListener("click", () => {
        window.open(`${config.url}/pay?amount=${amount}&email=${config.customerEmail}`, "_blank");
        overlay.remove();
      });
      grid.appendChild(btn);
    });
    modal.appendChild(grid);

    const close = document.createElement("button");
    close.textContent = "Cancel";
    close.style.cssText = `
      width: 100%; padding: 10px; background: transparent;
      border: 1px solid #555; border-radius: 8px; color: inherit;
      cursor: pointer; font-size: 14px;
    `;
    close.addEventListener("click", () => overlay.remove());
    modal.appendChild(close);

    overlay.appendChild(modal);
    overlay.addEventListener("click", (e) => {
      if (e.target === overlay) overlay.remove();
    });
    document.body.appendChild(overlay);
  }

  // Auto-inject button
  const container = script.parentElement;
  if (container) {
    container.appendChild(createButton());
  }
})();
