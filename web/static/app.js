// CodeBridge Frontend Application
class CodeBridge {
    constructor() {
        this.eventSource = null;
        this.isTranslating = false;
        this.currentTargetLang = 'javascript';
        this.sections = {
            explanation: null,
            notes: null,
            code: null
        };
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadLanguages();
        this.setupExamples();
    }

    setupEventListeners() {
        const translateBtn = document.getElementById('translateBtn');
        const clearBtn = document.getElementById('clearBtn');
        const swapBtn = document.getElementById('swapBtn');
        const copyBtn = document.getElementById('copyOutputBtn');
        const exampleSelect = document.getElementById('exampleSelect');
        const targetLangSelect = document.getElementById('targetLang');

        translateBtn.addEventListener('click', () => this.translate());
        clearBtn.addEventListener('click', () => this.clearAll());
        swapBtn.addEventListener('click', () => this.swapLanguages());
        copyBtn.addEventListener('click', () => this.copyOutput());
        exampleSelect.addEventListener('change', (e) => this.loadExample(e.target.value));
        targetLangSelect.addEventListener('change', (e) => this.currentTargetLang = e.target.value);

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                this.translate();
            }
        });
    }

    loadLanguages() {
        const languages = [
            { value: 'javascript', label: 'JavaScript' },
            { value: 'typescript', label: 'TypeScript' },
            { value: 'python', label: 'Python' },
            { value: 'go', label: 'Go' },
            { value: 'rust', label: 'Rust' },
            { value: 'java', label: 'Java' },
            { value: 'csharp', label: 'C#' },
            { value: 'cpp', label: 'C++' },
            { value: 'php', label: 'PHP' },
            { value: 'ruby', label: 'Ruby' },
            { value: 'swift', label: 'Swift' },
            { value: 'kotlin', label: 'Kotlin' }
        ];

        const sourceLangSelect = document.getElementById('sourceLang');
        const targetLangSelect = document.getElementById('targetLang');

        languages.forEach(lang => {
            sourceLangSelect.add(new Option(lang.label, lang.value));
            targetLangSelect.add(new Option(lang.label, lang.value));
        });

        // Set defaults
        sourceLangSelect.value = 'python';
        targetLangSelect.value = 'javascript';
    }

    setupExamples() {
        const examples = {
            'fibonacci': {
                lang: 'python',
                code: `def fibonacci(n):
    """Calculate the nth Fibonacci number."""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

# Example usage
for i in range(10):
    print(f"F({i}) = {fibonacci(i)}")`
            },
            'quicksort': {
                lang: 'javascript',
                code: `function quickSort(arr) {
  if (arr.length <= 1) return arr;
  
  const pivot = arr[Math.floor(arr.length / 2)];
  const left = arr.filter(x => x < pivot);
  const middle = arr.filter(x => x === pivot);
  const right = arr.filter(x => x > pivot);
  
  return [...quickSort(left), ...middle, ...quickSort(right)];
}

console.log(quickSort([3, 6, 8, 10, 1, 2, 1]));`
            },
            'http-server': {
                lang: 'go',
                code: `package main

import (
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}`
            }
        };

        this.examples = examples;
    }

    loadExample(exampleKey) {
        if (!exampleKey) return;
        
        const example = this.examples[exampleKey];
        if (example) {
            document.getElementById('sourceCode').value = example.code;
            document.getElementById('sourceLang').value = example.lang;
        }
    }

    async translate() {
        if (this.isTranslating) {
            this.showNotification('Translation already in progress', 'warning');
            return;
        }

        const sourceCode = document.getElementById('sourceCode').value.trim();
        const sourceLang = document.getElementById('sourceLang').value;
        const targetLang = document.getElementById('targetLang').value;

        if (!sourceCode) {
            this.showNotification('Please enter source code', 'error');
            return;
        }

        if (sourceLang === targetLang) {
            this.showNotification('Source and target languages must be different', 'error');
            return;
        }

        this.currentTargetLang = targetLang;
        this.isTranslating = true;
        this.updateUITranslating(true);
        this.clearOutput();

        try {
            const response = await fetch('/translate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    code: sourceCode,
                    source_language: sourceLang,
                    target_language: targetLang
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.startStreaming(data.id);

        } catch (error) {
            this.showNotification(`Translation failed: ${error.message}`, 'error');
            this.isTranslating = false;
            this.updateUITranslating(false);
        }
    }

    startStreaming(jobId) {
        const outputContainer = document.getElementById('outputContainer');
        const statusEl = document.getElementById('streamStatus');

        // Store sections as they arrive
        this.sections = {
            explanation: '',
            notes: '',
            code: ''
        };

        const timeoutId = setTimeout(() => {
            if (this.isTranslating) {
                this.showNotification('Stream timeout - please try again', 'error');
                if (this.eventSource) {
                    this.eventSource.close();
                }
                this.isTranslating = false;
                this.updateUITranslating(false);
                statusEl.textContent = 'Timeout';
                statusEl.className = 'status error';
            }
        }, 120000);

        this.eventSource = new EventSource(`/translate/stream/${jobId}`);

        this.eventSource.onopen = () => {
            console.log('Stream connected');
            statusEl.textContent = 'Connected, streaming...';
            statusEl.className = 'status streaming';
        };

        this.eventSource.onmessage = (event) => {
            clearTimeout(timeoutId);

            if (event.data === '[DONE]') {
                console.log('Stream end signal received');
                this.eventSource.close();
                this.isTranslating = false;
                this.updateUITranslating(false);
                statusEl.textContent = 'Translation completed';
                statusEl.className = 'status success';
                this.showNotification('Translation completed successfully!', 'success');
                return;
            }

            if (event.data.startsWith('ERROR:')) {
                clearTimeout(timeoutId);
                this.showNotification(event.data, 'error');
                this.eventSource.close();
                this.isTranslating = false;
                this.updateUITranslating(false);
                statusEl.textContent = 'Translation failed';
                statusEl.className = 'status error';
                return;
            }

            // Parse JSON chunk
            try {
                const chunk = JSON.parse(event.data);

                // Update the appropriate section
                if (chunk.type === 'explanation') {
                    this.sections.explanation = chunk.content;
                } else if (chunk.type === 'notes') {
                    this.sections.notes = chunk.content;
                } else if (chunk.type === 'code') {
                    this.sections.code = chunk.content;
                }

                // Render with streaming effect
                this.renderSections();

            } catch (e) {
                console.error('Failed to parse chunk:', e, event.data);
            }
        };

        this.eventSource.onerror = (error) => {
            clearTimeout(timeoutId);
            console.error('SSE Error:', error);
            this.showNotification('Connection error - please try again', 'error');
            if (this.eventSource) {
                this.eventSource.close();
            }
            this.isTranslating = false;
            this.updateUITranslating(false);
            statusEl.textContent = 'Connection error';
            statusEl.className = 'status error';
        };

        statusEl.textContent = 'Connecting...';
        statusEl.className = 'status streaming';
    }

    renderSections() {
        const outputContainer = document.getElementById('outputContainer');
        let html = '';

        // Render explanation if available
        if (this.sections.explanation) {
            html += `<div class="section explanation-section">
            <h3 class="section-header">📖 Explanation</h3>
            <div class="section-content">${this.formatText(this.sections.explanation)}</div>
        </div>`;
        }

        // Render notes if available
        if (this.sections.notes) {
            html += `<div class="section notes-section">
            <h3 class="section-header">💡 Translation Notes</h3>
            <div class="section-content">${this.formatNotes(this.sections.notes)}</div>
        </div>`;
        }

        // Render code if available
        if (this.sections.code) {
            const lang = this.getPrismLanguage(this.currentTargetLang);
            html += `<div class="section code-section">
            <h3 class="section-header">✨ Translated Code</h3>
            <pre class="line-numbers"><code class="language-${lang}">${this.escapeHtml(this.sections.code)}</code></pre>
        </div>`;
        }

        // Show loading indicator if not all sections are present
        const sectionsCount = [this.sections.explanation, this.sections.notes, this.sections.code].filter(Boolean).length;
        if (sectionsCount < 3 && this.isTranslating) {
            html += `<div class="streaming-indicator">
            <div class="spinner"></div>
            <span>Loading sections... (${sectionsCount}/3)</span>
        </div>`;
        }

        outputContainer.innerHTML = html || '<div class="loading-text">Waiting for response...</div>';

        // Apply syntax highlighting
        if (this.sections.code) {
            requestAnimationFrame(() => {
                outputContainer.querySelectorAll('pre code').forEach((block) => {
                    try {
                        if (window.Prism) {
                            Prism.highlightElement(block);
                        }
                    } catch (e) {
                        console.warn('Prism highlighting failed:', e);
                    }
                });

                // Scroll to bottom after highlighting is complete
                outputContainer.scrollTop = outputContainer.scrollHeight;
            });
        } else {
            // Scroll to bottom even without syntax highlighting
            requestAnimationFrame(() => {
                outputContainer.scrollTop = outputContainer.scrollHeight;
            });
        }
    }

    formatText(text) {
        const paragraphs = text
            .split(/\n\s*\n/)
            .filter(p => p.trim())
            .map(p => p.replace(/\n/g, ' ').trim());

        if (paragraphs.length === 0) {
            return `<p>${this.escapeHtml(text)}</p>`;
        }

        return paragraphs
            .map(p => `<p>${this.escapeHtml(p)}</p>`)
            .join('');
    }

    formatNotes(text) {
        const lines = text.split('\n').filter(line => line.trim());

        if (lines.length === 0) {
            return `<p>${this.escapeHtml(text)}</p>`;
        }

        let html = '<ul class="notes-list">';

        lines.forEach(line => {
            const cleaned = line
                .replace(/^[-*•]\s+/, '')
                .replace(/^\d+\.\s+/, '')
                .trim();

            if (cleaned) {
                html += `<li>${this.escapeHtml(cleaned)}</li>`;
            }
        });

        html += '</ul>';
        return html;
    }

    copyOutput() {
        if (this.sections.code) {
            navigator.clipboard.writeText(this.sections.code).then(() => {
                this.showNotification('Code copied to clipboard!', 'success');
            }).catch(() => {
                this.showNotification('Failed to copy code', 'error');
            });
        } else {
            this.showNotification('No code to copy yet', 'warning');
        }
    }

    getPrismLanguage(lang) {
        const langMap = {
            'javascript': 'javascript',
            'typescript': 'typescript',
            'python': 'python',
            'go': 'go',
            'rust': 'rust',
            'java': 'java',
            'csharp': 'csharp',
            'cpp': 'cpp',
            'php': 'php',
            'ruby': 'ruby',
            'swift': 'swift',
            'kotlin': 'kotlin'
        };
        return langMap[lang] || 'javascript';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    updateUITranslating(translating) {
        const translateBtn = document.getElementById('translateBtn');

        translateBtn.disabled = translating;

        if (translating) {
            translateBtn.innerHTML = '<span class="spinner"></span><span>Translating...</span>';
        } else {
            translateBtn.innerHTML = '<span>Translate</span>';
        }
    }


    clearAll() {
        document.getElementById('sourceCode').value = '';
        this.clearOutput();
        document.getElementById('exampleSelect').value = '';
    }

    clearOutput() {
        this.sections = {
            explanation: null,
            notes: null,
            code: null
        };
        document.getElementById('outputContainer').innerHTML = '<div class="placeholder-text">Translation result will appear here...</div>';
        const statusEl = document.getElementById('streamStatus');
        statusEl.textContent = 'Ready';
        statusEl.className = 'status';
    }

    swapLanguages() {
        const sourceLang = document.getElementById('sourceLang');
        const targetLang = document.getElementById('targetLang');
        const sourceCode = document.getElementById('sourceCode');

        // Swap language selections
        const tempLang = sourceLang.value;
        sourceLang.value = targetLang.value;
        targetLang.value = tempLang;

        // Clear output when swapping
        this.clearOutput();
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.classList.add('show');
        }, 10);

        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new CodeBridge();
});
