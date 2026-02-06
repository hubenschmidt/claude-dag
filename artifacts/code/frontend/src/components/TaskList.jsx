const FILTER_OPTIONS = [
  { label: "All", value: "" },
  { label: "Todo", value: "todo" },
  { label: "In Progress", value: "in_progress" },
  { label: "Done", value: "done" },
];

const TaskList = ({ tasks, filter, onFilterChange, onUpdate, onDelete }) => {
  if (!tasks.length && !filter) {
    return <p className="empty">No tasks yet. Add one above.</p>;
  }

  if (!tasks.length && filter) {
    return (
      <div>
        <FilterBar filter={filter} onChange={onFilterChange} />
        <p className="empty">No tasks match the "{filter}" filter.</p>
      </div>
    );
  }

  return (
    <div>
      <FilterBar filter={filter} onChange={onFilterChange} />
      <table className="task-table">
        <thead>
          <tr>
            <th>Title</th>
            <th>Description</th>
            <th>Status</th>
            <th>Created</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((task) => (
            <TaskItem
              key={task.id}
              task={task}
              onUpdate={onUpdate}
              onDelete={onDelete}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

const FilterBar = ({ filter, onChange }) => (
  <div className="filter-bar">
    {FILTER_OPTIONS.map((opt) => (
      <button
        key={opt.value}
        className={filter === opt.value ? "active" : ""}
        onClick={() => onChange(opt.value)}
      >
        {opt.label}
      </button>
    ))}
  </div>
);

window.TaskList = TaskList;
