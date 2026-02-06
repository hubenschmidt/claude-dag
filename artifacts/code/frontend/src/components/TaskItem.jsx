const API_BASE = "http://localhost:8080";

const STATUS_CYCLE = { todo: "in_progress", in_progress: "done", done: "todo" };

const TaskItem = ({ task, onUpdate, onDelete }) => {
  const [deleting, setDeleting] = React.useState(false);

  const handleStatusToggle = async () => {
    const nextStatus = STATUS_CYCLE[task.status];
    try {
      const res = await fetch(`${API_BASE}/tasks/${task.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: nextStatus }),
      });
      if (!res.ok) return;
      const updated = await res.json();
      onUpdate(updated);
    } catch (_) {
      /* silently fail — user can retry */
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      const res = await fetch(`${API_BASE}/tasks/${task.id}`, { method: "DELETE" });
      if (!res.ok) { setDeleting(false); return; }
      onDelete(task.id);
    } catch (_) {
      setDeleting(false);
    }
  };

  const created = new Date(task.created_at).toLocaleDateString();

  return (
    <tr>
      <td style={{ maxWidth: "200px" }}>{task.title}</td>
      <td style={{ maxWidth: "240px", color: "#666", fontSize: "0.85rem" }}>
        {task.description || "—"}
      </td>
      <td>
        <button
          className={`status-badge ${task.status}`}
          onClick={handleStatusToggle}
          title={`Click to change to ${STATUS_CYCLE[task.status]}`}
        >
          {task.status}
        </button>
      </td>
      <td style={{ fontSize: "0.8rem", color: "#888" }}>{created}</td>
      <td>
        <button className="btn-delete" onClick={handleDelete} disabled={deleting}>
          {deleting ? "…" : "Delete"}
        </button>
      </td>
    </tr>
  );
};

window.TaskItem = TaskItem;
