package cloudtasks

import "testing"

func TestTaskName(t *testing.T) {
	ct := &CloudTasks{queue: "projects/project/locations/location/queues/queue"}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "full resource name",
			in:   "projects/project/locations/location/queues/queue/tasks/task-id",
			want: "projects/project/locations/location/queues/queue/tasks/task-id",
		},
		{
			name: "full resource name from another queue",
			in:   "projects/project/locations/location/queues/old-queue/tasks/task-id",
			want: "projects/project/locations/location/queues/old-queue/tasks/task-id",
		},
		{
			name: "legacy bare task ID",
			in:   "task-id",
			want: "projects/project/locations/location/queues/queue/tasks/task-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ct.taskName(tt.in); got != tt.want {
				t.Fatalf("taskName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
