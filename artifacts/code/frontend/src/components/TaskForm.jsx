const TaskForm = ({ onCreated }) => {
  const [title, setTitle] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!title.trim()) return;

    setSubmitting(true);
    setError(null);

    try {
      const body = { title: title.trim() };
      if (description.trim()) body.description = description.trim();

      const res = await fetch("http://localhost:8080/tasks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const data = await res.json();
        setError(data.error || "Failed to create task");
        setSubmitting(false);
        return;
      }

      const created = await res.json();
      onCreated(created);
      setTitle("");
      setDescription("");
    } catch (_) {
      setError("Network error — is the server running?");
    }
    setSubmitting(false);
  };

  return (
    <form className="task-form" onSubmit={handleSubmit}>
      <input
        type="text"
        placeholder="Task title (required)"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        maxLength={255}
        required
      />
      <textarea
        placeholder="Description (optional)"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        maxLength={4096}
      />
      {error && <div className="error">{error}</div>}
      <button type="submit" disabled={submitting || !title.trim()}>
        {submitting ? "Adding…" : "Add Task"}
      </button>
    </form>
  );
};

window.TaskForm = TaskForm;
