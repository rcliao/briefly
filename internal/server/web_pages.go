package server

import (
	"html/template"
	"net/http"
	"os"
)

// getPostHogConfig returns PostHog configuration
func (s *Server) getPostHogConfig() (apiKey string, host string) {
	apiKey = os.Getenv("POSTHOG_API_KEY")
	host = os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://app.posthog.com"
	}
	return
}

// handleThemesPage handles GET /themes (HTML page)
func (s *Server) handleThemesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	apiKey, host := s.getPostHogConfig()

	tmpl := template.Must(template.New("themes").Parse(themesPageTemplate))
	data := map[string]interface{}{
		"PostHogAPIKey":  apiKey,
		"PostHogHost":    host,
		"PostHogEnabled": apiKey != "",
	}

	if err := tmpl.Execute(w, data); err != nil {
		s.log.Error("Failed to render themes page", "error", err)
	}
}

// handleSubmitPage handles GET /submit (HTML page)
func (s *Server) handleSubmitPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	apiKey, host := s.getPostHogConfig()

	tmpl := template.Must(template.New("submit").Parse(submitPageTemplate))
	data := map[string]interface{}{
		"PostHogAPIKey":  apiKey,
		"PostHogHost":    host,
		"PostHogEnabled": apiKey != "",
	}

	if err := tmpl.Execute(w, data); err != nil {
		s.log.Error("Failed to render submit page", "error", err)
	}
}

const themesPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Theme Management - Briefly</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2/dist/tailwind.min.css" rel="stylesheet">
    {{if .PostHogEnabled}}
    <script>
        !function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
        posthog.init('{{.PostHogAPIKey}}', {api_host: '{{.PostHogHost}}'})
    </script>
    {{end}}
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm border-b">
        <div class="container mx-auto px-4 py-4">
            <div class="flex items-center justify-between">
                <a href="/" class="text-2xl font-bold text-blue-600">Briefly</a>
                <div class="space-x-4">
                    <a href="/themes" class="text-blue-600 font-semibold">Themes</a>
                    <a href="/submit" class="text-gray-600 hover:text-blue-600">Submit URL</a>
                </div>
            </div>
        </div>
    </nav>

    <div class="container mx-auto px-4 py-8">
        <div class="max-w-6xl mx-auto">
            <div class="flex justify-between items-center mb-8">
                <h1 class="text-3xl font-bold text-gray-900">Theme Management</h1>
                <button onclick="showAddThemeModal()" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
                    + Add Theme
                </button>
            </div>

            <div id="themes-container" class="grid gap-4">
                <div class="text-center py-8 text-gray-500">Loading themes...</div>
            </div>
        </div>
    </div>

    <div id="addThemeModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
        <div class="bg-white rounded-lg p-8 max-w-md w-full mx-4">
            <h2 class="text-2xl font-bold mb-4">Add New Theme</h2>
            <form id="addThemeForm" class="space-y-4">
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
                    <input type="text" name="name" required
                           class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">Description</label>
                    <textarea name="description" rows="3"
                              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"></textarea>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">Keywords (comma-separated)</label>
                    <input type="text" name="keywords" placeholder="AI, machine learning, ML"
                           class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                </div>
                <div class="flex items-center">
                    <input type="checkbox" name="enabled" checked class="mr-2">
                    <label class="text-sm text-gray-700">Enabled</label>
                </div>
                <div class="flex space-x-3">
                    <button type="submit" class="flex-1 bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700">
                        Add Theme
                    </button>
                    <button type="button" onclick="closeAddThemeModal()"
                            class="flex-1 bg-gray-200 text-gray-700 py-2 rounded-lg hover:bg-gray-300">
                        Cancel
                    </button>
                </div>
            </form>
        </div>
    </div>

    <script src="/static/themes.js"></script>
    <script>
        // Inline fallback if static file not available
        if (typeof loadThemes === 'undefined') {
            async function loadThemes() {
                try {
                    const response = await fetch('/api/themes');
                    const data = await response.json();
                    const container = document.getElementById('themes-container');

                    if (data.themes.length === 0) {
                        container.innerHTML = '<div class="text-center py-8 text-gray-500">No themes yet. Add your first theme!</div>';
                        return;
                    }

                    container.innerHTML = data.themes.map(theme =>
                        '<div class="bg-white rounded-lg shadow p-6">' +
                        '<div class="flex justify-between items-start">' +
                        '<div class="flex-1">' +
                        '<div class="flex items-center gap-3 mb-2">' +
                        '<h3 class="text-xl font-semibold">' + theme.name + '</h3>' +
                        '<span class="px-2 py-1 text-xs rounded ' + (theme.enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800') + '">' +
                        (theme.enabled ? 'Enabled' : 'Disabled') +
                        '</span></div>' +
                        '<p class="text-gray-600 mb-3">' + (theme.description || 'No description') + '</p>' +
                        '<div class="flex flex-wrap gap-2">' +
                        (theme.keywords || []).map(kw => '<span class="px-2 py-1 bg-blue-50 text-blue-700 text-sm rounded">' + kw + '</span>').join('') +
                        '</div></div>' +
                        '<button onclick="toggleTheme(\'' + theme.id + '\', ' + !theme.enabled + ')" class="ml-4 text-sm text-blue-600 hover:text-blue-800">' +
                        (theme.enabled ? 'Disable' : 'Enable') +
                        '</button></div></div>'
                    ).join('');

                    {{if .PostHogEnabled}}
                    if (window.posthog) posthog.capture('themes_page_viewed', { theme_count: data.themes.length });
                    {{end}}
                } catch (error) {
                    console.error('Failed to load themes:', error);
                }
            }

            document.getElementById('addThemeForm').addEventListener('submit', async (e) => {
                e.preventDefault();
                const formData = new FormData(e.target);
                const keywords = formData.get('keywords').split(',').map(k => k.trim()).filter(k => k);

                try {
                    const response = await fetch('/api/themes', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            name: formData.get('name'),
                            description: formData.get('description'),
                            keywords: keywords,
                            enabled: formData.get('enabled') === 'on'
                        })
                    });

                    if (response.ok) {
                        closeAddThemeModal();
                        loadThemes();
                        {{if .PostHogEnabled}}
                        if (window.posthog) posthog.capture('theme_created', { theme_name: formData.get('name') });
                        {{end}}
                    } else {
                        const error = await response.json();
                        alert('Failed to add theme: ' + error.error.message);
                    }
                } catch (error) {
                    alert('Failed to add theme: ' + error.message);
                }
            });

            async function toggleTheme(id, enabled) {
                try {
                    const response = await fetch('/api/themes/' + id, {
                        method: 'PATCH',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ enabled })
                    });

                    if (response.ok) {
                        loadThemes();
                        {{if .PostHogEnabled}}
                        if (window.posthog) posthog.capture('theme_toggled', { theme_id: id, enabled });
                        {{end}}
                    } else {
                        alert('Failed to update theme');
                    }
                } catch (error) {
                    alert('Failed to update theme: ' + error.message);
                }
            }

            function showAddThemeModal() {
                document.getElementById('addThemeModal').classList.remove('hidden');
            }

            function closeAddThemeModal() {
                document.getElementById('addThemeModal').classList.add('hidden');
                document.getElementById('addThemeForm').reset();
            }

            loadThemes();
        }
    </script>
</body>
</html>`

const submitPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Submit URLs - Briefly</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2/dist/tailwind.min.css" rel="stylesheet">
    {{if .PostHogEnabled}}
    <script>
        !function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
        posthog.init('{{.PostHogAPIKey}}', {api_host: '{{.PostHogHost}}'})
    </script>
    {{end}}
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm border-b">
        <div class="container mx-auto px-4 py-4">
            <div class="flex items-center justify-between">
                <a href="/" class="text-2xl font-bold text-blue-600">Briefly</a>
                <div class="space-x-4">
                    <a href="/themes" class="text-gray-600 hover:text-blue-600">Themes</a>
                    <a href="/submit" class="text-blue-600 font-semibold">Submit URL</a>
                </div>
            </div>
        </div>
    </nav>

    <div class="container mx-auto px-4 py-8">
        <div class="max-w-2xl mx-auto">
            <h1 class="text-3xl font-bold text-gray-900 mb-8">Submit URLs for Digest</h1>

            <div class="bg-white rounded-lg shadow p-6 mb-8">
                <form id="submitForm" class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-2">
                            URLs (one per line)
                        </label>
                        <textarea id="urlsInput" name="urls" rows="6" required
                                  placeholder="https://example.com/article1
https://example.com/article2"
                                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"></textarea>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">
                            Your name/email (optional)
                        </label>
                        <input type="text" name="submitted_by" placeholder="john@example.com"
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                    </div>
                    <button type="submit" class="w-full bg-blue-600 text-white py-3 rounded-lg hover:bg-blue-700 font-semibold">
                        Submit URLs
                    </button>
                </form>
                <div id="submitResult" class="mt-4 hidden"></div>
            </div>

            <div class="bg-white rounded-lg shadow p-6">
                <h2 class="text-xl font-semibold mb-4">Recently Submitted URLs</h2>
                <div id="urlsList">
                    <div class="text-center py-4 text-gray-500">Loading...</div>
                </div>
            </div>
        </div>
    </div>

    <script>
        async function loadURLs() {
            try {
                const response = await fetch('/api/manual-urls');
                const data = await response.json();
                const container = document.getElementById('urlsList');

                if (data.urls.length === 0) {
                    container.innerHTML = '<div class="text-center py-4 text-gray-500">No URLs submitted yet</div>';
                    return;
                }

                container.innerHTML = data.urls.map(url => {
                    const statusColors = { 'pending': 'bg-yellow-100 text-yellow-800', 'processing': 'bg-blue-100 text-blue-800', 'processed': 'bg-green-100 text-green-800', 'failed': 'bg-red-100 text-red-800' };
                    const statusColor = statusColors[url.status] || 'bg-gray-100 text-gray-800';
                    const retryBtn = url.status === 'failed' ? '<button onclick="retryURL(\'' + url.id + '\')" class="ml-2 text-xs text-blue-600 hover:text-blue-800">Retry</button>' : '';

                    return '<div class="border-b py-3 last:border-b-0"><div class="flex justify-between items-start"><div class="flex-1">' +
                           '<div class="text-sm font-medium text-gray-900 truncate mb-1">' + url.url + '</div>' +
                           '<div class="flex items-center gap-3 text-xs text-gray-500">' +
                           '<span class="px-2 py-1 rounded ' + statusColor + '">' + url.status + '</span>' +
                           (url.submitted_by ? '<span>by ' + url.submitted_by + '</span>' : '') +
                           '<span>' + new Date(url.created_at).toLocaleDateString() + '</span>' +
                           '</div></div>' + retryBtn + '</div></div>';
                }).join('');

                {{if .PostHogEnabled}}
                if (window.posthog) posthog.capture('submit_page_viewed', { url_count: data.urls.length });
                {{end}}
            } catch (error) {
                console.error('Failed to load URLs:', error);
            }
        }

        document.getElementById('submitForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const urlsText = formData.get('urls');
            const urls = urlsText.split('\n').map(u => u.trim()).filter(u => u);

            const resultDiv = document.getElementById('submitResult');
            resultDiv.className = 'mt-4 p-4 rounded-lg';

            try {
                const response = await fetch('/api/manual-urls', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        urls: urls,
                        submitted_by: formData.get('submitted_by') || ''
                    })
                });

                const data = await response.json();

                if (response.ok) {
                    resultDiv.classList.remove('hidden');
                    resultDiv.classList.add('bg-green-50', 'text-green-800');
                    resultDiv.innerHTML = '<strong>Success!</strong> ' + data.total + ' URL(s) submitted';

                    document.getElementById('urlsInput').value = '';
                    loadURLs();

                    {{if .PostHogEnabled}}
                    if (window.posthog) posthog.capture('urls_submitted', { url_count: data.total });
                    {{end}}

                    setTimeout(() => { resultDiv.classList.add('hidden'); }, 5000);
                } else {
                    resultDiv.classList.remove('hidden');
                    resultDiv.classList.add('bg-red-50', 'text-red-800');
                    resultDiv.innerHTML = '<strong>Error!</strong> ' + (data.error && data.error.message ? data.error.message : 'Failed to submit URLs');
                }
            } catch (error) {
                resultDiv.classList.remove('hidden');
                resultDiv.classList.add('bg-red-50', 'text-red-800');
                resultDiv.innerHTML = '<strong>Error!</strong> ' + error.message;
            }
        });

        async function retryURL(id) {
            try {
                const response = await fetch('/api/manual-urls/' + id + '/retry', { method: 'POST' });
                if (response.ok) {
                    loadURLs();
                    {{if .PostHogEnabled}}
                    if (window.posthog) posthog.capture('url_retried', { url_id: id });
                    {{end}}
                } else {
                    alert('Failed to retry URL');
                }
            } catch (error) {
                alert('Failed to retry URL: ' + error.message);
            }
        }

        loadURLs();
    </script>
</body>
</html>`
