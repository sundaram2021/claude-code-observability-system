// Claude Code Agent Task Simulator
const Simulator = {
  init() {
    const btnSim = document.getElementById('btn-simulate');
    const modal = document.getElementById('sim-modal');
    const btnClose = document.getElementById('sim-close');
    const form = document.getElementById('sim-form');

    btnSim.addEventListener('click', () => {
      modal.classList.remove('hidden');
    });

    btnClose.addEventListener('click', () => {
      modal.classList.add('hidden');
    });

    modal.addEventListener('click', (e) => {
      if (e.target === modal) modal.classList.add('hidden');
    });

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const prompt = document.getElementById('sim-prompt').value.trim();
      const model = document.getElementById('sim-model').value;
      const btnSubmit = document.getElementById('btn-run-sim');

      if (!prompt) return;

      btnSubmit.disabled = true;
      btnSubmit.textContent = 'Running Agent...';

      try {
        await API.runAgent({
          prompt,
          model,
          simulate: true,
          project: 'claude-code-workspace',
        });
        modal.classList.add('hidden');
        await App.refreshAll();
      } catch (err) {
        alert('Simulation failed: ' + err.message);
      } finally {
        btnSubmit.disabled = false;
        btnSubmit.textContent = 'Execute Agent Task';
      }
    });
  },
};

