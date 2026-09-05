package filter

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// DedupManager evita reprocessar vagas já enviadas.
type DedupManager struct {
	seen map[string]bool
	file string
	mu   sync.Mutex
}

// NewDedup cria um dedup manager. Se o arquivo existir, carrega os IDs.
func NewDedup(filePath string) *DedupManager {
	d := &DedupManager{
		seen: make(map[string]bool),
		file: filePath,
	}
	d.load()
	return d
}

// IsSeen verifica se uma vaga já foi processada.
func (d *DedupManager) IsSeen(jobID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.seen[jobID]
}

// MarkSeen marca uma vaga como processada e salva no disco.
func (d *DedupManager) MarkSeen(jobID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[jobID] = true
	d.save()
}

// FilterNew retorna apenas vagas que ainda não foram vistas.
func (d *DedupManager) FilterNew(jobs []linkedin.Job) []linkedin.Job {
	var newJobs []linkedin.Job
	for _, job := range jobs {
		if !d.IsSeen(job.ID) {
			newJobs = append(newJobs, job)
		}
	}
	return newJobs
}

// MarkAll marca todas as vagas como vistas.
func (d *DedupManager) MarkAll(jobs []linkedin.Job) {
	for _, job := range jobs {
		d.MarkSeen(job.ID)
	}
}

func (d *DedupManager) load() {
	data, err := os.ReadFile(d.file)
	if err != nil {
		return
	}
	json.Unmarshal(data, &d.seen)
}

func (d *DedupManager) save() {
	data, err := json.MarshalIndent(d.seen, "", " ")
	if err != nil {
		return
	}
	os.WriteFile(d.file, data, 0644)
}
