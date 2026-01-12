const port = import.meta.env.VITE_PORT || '5173'
const isWorktree = import.meta.env.VITE_PORT !== undefined

document.getElementById('port').textContent = port
document.getElementById('context').textContent = isWorktree
  ? `This is a worktree with a custom port configured via WT_INDEX.`
  : `This is the main repository (default port).`
