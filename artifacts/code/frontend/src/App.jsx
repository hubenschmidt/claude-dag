const { useState, useEffect } = React;

const API_URL = "http://localhost:8080";

const App = () => {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState("");

  const fetchTasks = async (statusFilter) => {
    setLoading(true);
    setError(null);
    try {
      const qs = statusFilter ? `?status=${statusFilter}` : "";
      const res = await fetch(`${API_URL}/tasks${qs}`);
      if (!res.ok) throw new Error("Failed to fetch tasks");
      const data = await res.json();
      setTasks(data.tasks || []);
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchTasks(filter);
  }, [filter]);

  const handleCreated = (task) => {
    if (filter && task.status !== filter) return;
    setTasks((prev) => [...prev, task]);
  };

  const handleUpdate = (updated) => {
    if (filter && updated.status !== filter) {
      setTasks((prev) => prev.filter((t) => t.id !== updated.id));
      return;
    }
    setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)));
  };

  const handleDelete = (id) => {
    setTasks((prev) => prev.filter((t) => t.id !== id));
  };

  return (
    <div className="app">
      <h1>Task Manager</h1>
      <TaskForm onCreated={handleCreated} />
      {error && <div className="error">{error}</div>}
      {loading ? (
        <p className="loading">Loading tasks…</p>
      ) : (
        <TaskList
          tasks={tasks}
          filter={filter}
          onFilterChange={setFilter}
          onUpdate={handleUpdate}
          onDelete={handleDelete}
        />
      )}
    </div>
  );
};

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(<App />);
