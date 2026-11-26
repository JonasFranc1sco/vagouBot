package filter

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// DedupManager evita reprocessar vagas já enviadas ou avaliadas.
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

// jobSignatureKey gera uma chave normalizada "sig:empresa|titulo" para barrar
// vagas republicadas (reposted) pelo LinkedIn com novo ID numérico.
func jobSignatureKey(job linkedin.Job) string {
	comp := strings.ToLower(strings.TrimSpace(job.Company))
	title := strings.ToLower(strings.TrimSpace(job.Title))
	if comp == "" || title == "" {
		return ""
	}
	return "sig:" + comp + "|" + title
}

// IsSeen verifica se uma vaga já foi processada pelo ID.
func (d *DedupManager) IsSeen(jobID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.seen[jobID]
}

// IsJobSeen verifica se a vaga já foi vista pelo ID ou pela assinatura (empresa + título).
func (d *DedupManager) IsJobSeen(job linkedin.Job) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if job.ID != "" && d.seen[job.ID] {
		return true
	}
	if sig := jobSignatureKey(job); sig != "" && d.seen[sig] {
		return true
	}
	return false
}

// MarkSeen marca um ID de vaga como processado e salva no disco.
func (d *DedupManager) MarkSeen(jobID string) {
	if jobID == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[jobID] = true
	d.save()
}

// MarkSeenJob marca o ID e a assinatura (empresa + título) da vaga como vistos.
func (d *DedupManager) MarkSeenJob(job linkedin.Job) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if job.ID != "" {
		d.seen[job.ID] = true
	}
	if sig := jobSignatureKey(job); sig != "" {
		d.seen[sig] = true
	}
	d.save()
}

// FilterNew retorna apenas vagas que ainda não foram vistas (nem no histórico
// nem repetidas dentro do próprio lote atual).
func (d *DedupManager) FilterNew(jobs []linkedin.Job) []linkedin.Job {
	d.mu.Lock()
	defer d.mu.Unlock()

	var newJobs []linkedin.Job
	batchSeen := make(map[string]bool)

	for _, job := range jobs {
		if job.ID == "" {
			continue
		}
		if d.seen[job.ID] || batchSeen[job.ID] {
			continue
		}

		sig := jobSignatureKey(job)
		if sig != "" && (d.seen[sig] || batchSeen[sig]) {
			continue
		}

		batchSeen[job.ID] = true
		if sig != "" {
			batchSeen[sig] = true
		}
		newJobs = append(newJobs, job)
	}
	return newJobs
}

// MarkAll marca todas as vagas do slice como vistas de uma só vez.
func (d *DedupManager) MarkAll(jobs []linkedin.Job) {
	if len(jobs) == 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, job := range jobs {
		if job.ID != "" {
			d.seen[job.ID] = true
		}
		if sig := jobSignatureKey(job); sig != "" {
			d.seen[sig] = true
		}
	}
	d.save()
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
