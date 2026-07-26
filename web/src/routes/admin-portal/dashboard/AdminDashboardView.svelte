<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';

	interface RecentActivity {
		name: string;
		regNo: string;
		activity: string;
		date: string;
		hours: string;
		status: string;
	}

	interface PendingCertificate {
		id: string;
		student: string;
		regNo: string;
		type: string;
		date: string;
		priority: string;
		file: string;
		credits: number;
		name: string;
	}

	interface RawActivity {
		Student?: { Name?: string };
		student?: { name?: string };
		StudentRollNo?: string;
		student_roll_no?: string;
		ActivityName?: string;
		activity_name?: string;
		CreatedAt?: string;
		created_at?: string;
		Credits?: number;
		credits?: number;
		Status?: string;
		status?: string;
	}

	interface RawCertificate {
		id: number | string;
		student?: { name?: string };
		student_roll_no: string;
		activity_category: string;
		created_at: string;
		proof_url?: string;
		credits: number;
		activity_name: string;
		status: string;
	}

	// Props using Svelte 5 runes
	let {
		onTabChange
	}: {
		onTabChange: (tab: string) => void;
	} = $props();

	// Active State for Interactive elements
	let stats = $state({
		total_students: 0,
		active_students: 0,
		pending_reviews: 0,
		average_credits: 0,
		verification_rate: 0
	});

	// Mock Certificate Data State -> Now dynamic
	let pendingCertificates = $state<PendingCertificate[]>([]);

	// Recent Student Activities (Read-only list, matching image)
	let recentActivities = $state<RecentActivity[]>([]);

	onMount(async () => {
		const token = localStorage.getItem('admin_token');
		if (!token) return;

		try {
			const headers = { Authorization: `Bearer ${token}` };

			// 1. Fetch Stats
			const statsRes = await fetch(`${API_BASE_URL}/api/admin/dashboard/stats`, { headers });
			if (statsRes.ok) {
				stats = await statsRes.json();
			}

			// 2. Fetch Recent Activities
			const recentRes = await fetch(`${API_BASE_URL}/api/admin/dashboard/recent-activities`, {
				headers
			});
			if (recentRes.ok) {
				const data = await recentRes.json();
				recentActivities = (data.recent_activities || []).map((c: RawActivity) => ({
					name: c.Student?.Name || c.student?.name || 'Unknown',
					regNo: c.StudentRollNo || c.student_roll_no,
					activity: c.ActivityName || c.activity_name,
					date: new Date(c.CreatedAt || c.created_at || '').toLocaleDateString('en-GB'),
					hours: `${c.Credits || c.credits} cr`,
					status:
						(c.Status || c.status) === 'Approved'
							? 'Completed'
							: (c.Status || c.status) === 'Pending'
								? 'Pending Review'
								: 'Rejected'
				}));
			}

			// 3. Fetch Pending Certificates for the preview modal
			const certsRes = await fetch(`${API_BASE_URL}/api/admin/certificates`, { headers }); // Or wherever your queue endpoint is
			if (certsRes.ok) {
				const data = await certsRes.json();

				pendingCertificates = (data.certificates || [])
					.filter((c: RawCertificate) => c.status === 'Pending')
					.map((c: RawCertificate) => ({
						id: String(c.id),
						student: c.student?.name || 'Unknown',
						regNo: c.student_roll_no,
						type: c.activity_category,
						date: new Date(c.created_at).toLocaleDateString('en-GB', {
							day: '2-digit',
							month: 'short',
							year: 'numeric'
						}),
						priority: 'Normal', // Static for now
						file: c.proof_url || 'view_certificate',
						credits: c.credits,
						name: c.activity_name
					}));
			}
		} catch (e) {
			console.error('Failed to load dashboard data:', e);
		}
	});

	// Modal state
	let activeCertificate = $state<(typeof pendingCertificates)[0] | null>(null);
	let isModalOpen = $state(false);

	// Toasts state
	interface Toast {
		id: number;
		message: string;
		type: 'success' | 'danger';
	}
	let toasts = $state<Toast[]>([]);
	let nextToastId = 0;

	function triggerToast(message: string, type: 'success' | 'danger' = 'success') {
		const id = nextToastId++;
		toasts = [...toasts, { id, message, type }];
		setTimeout(() => {
			toasts = toasts.filter((t) => t.id !== id);
		}, 3000);
	}

	async function handleApprove(id: string) {
		const cert = pendingCertificates.find((c) => c.id === id);
		if (!cert) return;

		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${id}/approve`, {
				method: 'POST',
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				pendingCertificates = pendingCertificates.filter((c) => c.id !== id);
				stats.pending_reviews = Math.max(0, stats.pending_reviews - 1);
				triggerToast(`Approved certificate for ${cert.student} successfully!`);
				if (activeCertificate?.id === id) {
					closeModal();
				}
			} else {
				triggerToast('Failed to approve certificate', 'danger');
			}
		} catch {
			triggerToast('Failed to approve certificate', 'danger');
		}
	}

	async function handleReject(id: string) {
		const cert = pendingCertificates.find((c) => c.id === id);
		if (!cert) return;

		const reason = prompt('Reason for rejection:');
		if (reason === null) return;

		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${id}/reject`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ reason })
			});
			if (res.ok) {
				pendingCertificates = pendingCertificates.filter((c) => c.id !== id);
				stats.pending_reviews = Math.max(0, stats.pending_reviews - 1);
				triggerToast(`Rejected certificate for ${cert.student}.`, 'danger');
				if (activeCertificate?.id === id) {
					closeModal();
				}
			} else {
				triggerToast('Failed to reject certificate', 'danger');
			}
		} catch {
			triggerToast('Failed to reject certificate', 'danger');
		}
	}

	function openModal(cert: (typeof pendingCertificates)[0]) {
		activeCertificate = cert;
		isModalOpen = true;
	}

	async function downloadCertificate(cert: (typeof pendingCertificates)[0]) {
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${cert.id}/download`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});

			if (res.ok) {
				const blob = await res.blob();
				const contentDisposition = res.headers.get('content-disposition');
				let filename = cert.file || `Certificate_${cert.id}_${cert.regNo}`;
				if (contentDisposition) {
					const match = contentDisposition.match(/filename="?([^"]+)"?/);
					if (match && match[1]) {
						filename = match[1];
					}
				}
				const url = URL.createObjectURL(blob);
				const link = document.createElement('a');
				link.href = url;
				link.download = filename;
				document.body.appendChild(link);
				link.click();
				link.remove();
				URL.revokeObjectURL(url);
				triggerToast(`Downloaded submitted certificate file for ${cert.student}`);
				return;
			}
			triggerToast('Certificate file is no longer available on server', 'danger');
		} catch (e) {
			console.error('Failed to download certificate file from backend', e);
			triggerToast('Failed to download certificate file from server', 'danger');
		}
	}

	function closeModal() {
		isModalOpen = false;
		activeCertificate = null;
	}
</script>

<!-- Toast Container -->
<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
	{#each toasts as toast (toast.id)}
		<div
			transition:slide={{ duration: 150 }}
			class="p-4 rounded-xl border shadow-lg flex items-center justify-between gap-3 text-xs font-semibold font-sans transition-all {toast.type ===
			'success'
				? 'bg-emerald-50 border-emerald-200 text-emerald-800'
				: 'bg-rose-50 border-rose-200 text-rose-800'}"
		>
			<div class="flex items-center gap-2">
				{#if toast.type === 'success'}
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2.5"
						stroke="currentColor"
						class="w-4 h-4 text-emerald-600"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
					</svg>
				{:else}
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2.5"
						stroke="currentColor"
						class="w-4 h-4 text-rose-600"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
						/>
					</svg>
				{/if}
				<span>{toast.message}</span>
			</div>
		</div>
	{/each}
</div>

<!-- ==================== OVERVIEW CARDS ROW ==================== -->
<section
	class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5"
	aria-label="Overview Statistics"
>
	<!-- Card 1: Total Students -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.total_students}</span>
			<div class="p-2.5 rounded-lg bg-blue-50 text-blue-600 border border-blue-100">
				<!-- Users Icon -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide font-sans">Total Students</h3>
			<p class="text-[10px] font-bold text-slate-400 mt-1 uppercase tracking-wider">
				+2 this semester
			</p>
		</div>
	</div>

	<!-- Card 2: Pending Certificate Reviews -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.pending_reviews}</span>
			<div class="p-2.5 rounded-lg bg-amber-50 text-amber-600 border border-amber-100">
				<!-- File Icon -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide font-sans">
				Pending Certificate Reviews
			</h3>
			<p class="text-[10px] font-bold text-slate-400 mt-1 uppercase tracking-wider">
				{pendingCertificates.length > 0
					? `${pendingCertificates.filter((c) => c.priority === 'High').length} marked urgent`
					: 'All caught up!'}
			</p>
		</div>
	</div>

	<!-- Card 3: Total Activities -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.active_students}</span>
			<div class="p-2.5 rounded-lg bg-purple-50 text-purple-600 border border-purple-100">
				<!-- Wave Chart Icon -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide font-sans">Active Students</h3>
			<p class="text-[10px] font-bold text-slate-400 mt-1 uppercase tracking-wider">
				+12 this week
			</p>
		</div>
	</div>

	<!-- Card 4: Verification Rate -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900"
				>{(stats?.average_credits ?? 0).toFixed(1)}</span
			>
			<div class="p-2.5 rounded-lg bg-teal-50 text-teal-600 border border-teal-100">
				<!-- Shield Check Icon -->
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide font-sans">Average Credits</h3>
			<p class="text-[10px] font-bold text-slate-400 mt-1 uppercase tracking-wider">
				+4.2% from last batch
			</p>
		</div>
	</div>
</section>

<!-- ==================== MIDDLE SECTION GRID ==================== -->
<section class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
	<!-- Left Side: Recent Activity Submissions -->
	<div class="lg:col-span-8 bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
		<div class="p-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/20">
			<div>
				<h2 class="text-sm font-bold font-serif text-inst-navy">Recent Activity Submissions</h2>
			</div>
		</div>

		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr
						class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold text-slate-405 uppercase tracking-wider"
					>
						<th class="py-3 px-5">Student</th>
						<th class="py-3 px-5">Activity</th>
						<th class="py-3 px-5">Date</th>
						<th class="py-3 px-5">Hours</th>
						<th class="py-3 px-5">Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100 text-xs font-sans">
					{#each recentActivities as act}
						<tr class="hover:bg-slate-50/50 transition-colors">
							<td class="py-3.5 px-5">
								<div class="font-bold text-slate-800">{act.name}</div>
								<div class="text-[10px] text-slate-400 font-semibold uppercase">{act.regNo}</div>
							</td>
							<td class="py-3.5 px-5 font-semibold text-slate-700">{act.activity}</td>
							<td class="py-3.5 px-5 text-slate-500 font-medium">{act.date}</td>
							<td class="py-3.5 px-5 text-slate-800 font-extrabold">{act.hours}</td>
							<td class="py-3.5 px-5">
								<span
									class="inline-flex items-center px-2 py-1 rounded-md text-[10px] font-extrabold uppercase tracking-wide
									{act.status === 'Completed'
										? 'bg-emerald-55 text-emerald-700 border border-emerald-100'
										: act.status === 'In Progress'
											? 'bg-blue-50 text-blue-700 border border-blue-100'
											: 'bg-amber-50 text-amber-700 border border-amber-100'}"
								>
									{act.status}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

	<!-- Right Side: Quick Actions -->
	<div class="lg:col-span-4 bg-white border border-slate-200 rounded-xl p-5 shadow-xs space-y-4">
		<div>
			<h2 class="text-sm font-bold font-serif text-inst-navy">Quick Actions</h2>
		</div>

		<div class="grid grid-cols-2 gap-3.5 pt-2">
			<!-- Review Certificates (Accent Red box) -->
			<button
				onclick={() => onTabChange('Certificate Verification')}
				class="flex flex-col items-center justify-center p-4 rounded-xl bg-accent-red text-white hover:bg-accent-red/90 text-center shadow-xs transition duration-200 border border-transparent select-none focus:outline-none"
			>
				<div class="p-2 bg-white/10 rounded-lg mb-2">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M9 12.75 11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 0 1-1.043 3.296 3.745 3.745 0 0 1-3.296 1.043A3.745 3.745 0 0 1 12 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 0 1-3.296-1.043 3.745 3.745 0 0 1-1.043-3.296A3.745 3.745 0 0 1 3 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 0 1 1.043-3.296 3.746 3.746 0 0 1 3.296-1.043A3.746 3.746 0 0 1 12 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 0 1 3.296 1.043 3.746 3.746 0 0 1 1.043 3.296A3.745 3.745 0 0 1 21 12Z"
						/>
					</svg>
				</div>
				<span class="text-xs font-bold font-serif leading-tight">Review Certificates</span>
				<span class="text-[9px] font-semibold text-white/70 block mt-1">Verify pending</span>
			</button>

			<!-- Manage Students -->
			<button
				onclick={() => onTabChange('Student Management')}
				class="flex flex-col items-center justify-center p-4 rounded-xl bg-white text-slate-800 hover:bg-slate-50 text-center shadow-xs transition duration-200 border border-slate-200 select-none focus:outline-none"
			>
				<div class="p-2 bg-blue-50 text-blue-600 rounded-lg mb-2 border border-blue-100">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
						/>
					</svg>
				</div>
				<span class="text-xs font-bold font-serif text-slate-900 leading-tight"
					>Manage Students</span
				>
				<span class="text-[9px] font-semibold text-slate-400 block mt-1">Browse all students</span>
			</button>

			<!-- Browse Analytics -->
			<button
				onclick={() => onTabChange('Batch Analytics')}
				class="flex flex-col items-center justify-center p-4 rounded-xl bg-white text-slate-800 hover:bg-slate-50 text-center shadow-xs transition duration-200 border border-slate-200 select-none focus:outline-none"
			>
				<div class="p-2 bg-purple-50 text-purple-600 rounded-lg mb-2 border border-purple-100">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M10.5 6a7.5 7.5 0 1 0 7.5 7.5h-7.5V6Z"
						/>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M13.5 10.5H21a7.5 7.5 0 0 0-7.5-7.5v7.5Z"
						/>
					</svg>
				</div>
				<span class="text-xs font-bold font-serif text-slate-900 leading-tight"
					>Browse Analytics</span
				>
				<span class="text-[9px] font-semibold text-slate-400 block mt-1">View performances</span>
			</button>

			<!-- Generate Report -->
			<button
				onclick={() =>
					triggerToast('Cohort Activity Report generated and downloaded successfully!')}
				class="flex flex-col items-center justify-center p-4 rounded-xl bg-white text-slate-800 hover:bg-slate-50 text-center shadow-xs transition duration-200 border border-slate-200 select-none focus:outline-none"
			>
				<div class="p-2 bg-teal-50 text-teal-600 rounded-lg mb-2 border border-teal-100">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3"
						/>
					</svg>
				</div>
				<span class="text-xs font-bold font-serif text-slate-900 leading-tight"
					>Generate Report</span
				>
				<span class="text-[9px] font-semibold text-slate-400 block mt-1">Download records</span>
			</button>
		</div>
	</div>
</section>

<!-- ==================== STUDENT PROGRESS SUMMARY ==================== -->
<section class="bg-white border border-slate-200 p-5 rounded-xl shadow-xs space-y-4">
	<div>
		<h2 class="text-sm font-bold font-serif text-inst-navy">Student Progress Summary</h2>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-12 gap-6 items-center pt-2">
		<!-- Cohort Average Bar -->
		<div class="md:col-span-7 space-y-4">
			<div class="flex items-center justify-between">
				<div class="flex items-baseline gap-2">
					<span class="text-3xl font-extrabold text-inst-navy leading-none">142.5</span>
					<span class="text-[10px] font-bold text-slate-405 uppercase tracking-widest"
						>Average Credits Earned</span
					>
					<span class="text-slate-300 font-light">/</span>
					<span class="text-sm font-bold text-slate-500">200</span>
				</div>
				<span
					class="bg-emerald-50 text-emerald-700 text-[10px] font-extrabold uppercase px-2.5 py-1 rounded border border-emerald-100"
				>
					71.2% Average Progress
				</span>
			</div>

			<div class="space-y-1.5">
				<!-- Custom Progress Bar -->
				<div class="h-3 w-full bg-slate-100 rounded-full overflow-hidden relative">
					<div class="h-full bg-accent-red rounded-full" style="width: 71.2%"></div>
				</div>
				<div class="flex justify-between text-[9px] font-bold text-slate-400 px-1 font-sans">
					<span>0 Credits</span>
					<span>50</span>
					<span>100</span>
					<span>150</span>
					<span>200 (Target)</span>
				</div>
			</div>
		</div>

		<!-- Vertical Divider -->
		<div class="hidden md:block md:col-span-1 text-center text-slate-200">
			<div class="w-px h-16 bg-slate-200 mx-auto"></div>
		</div>

		<!-- Status Breakdown -->
		<div class="md:col-span-4 grid grid-cols-3 gap-2 text-center">
			<div class="p-2.5 rounded-lg bg-emerald-50/50 border border-emerald-100/50">
				<div class="text-lg font-extrabold text-emerald-700">8</div>
				<div class="text-[9px] font-bold text-slate-500 uppercase tracking-wide mt-0.5">
					Completed
				</div>
				<div class="text-[8px] font-semibold text-slate-400 mt-0.5">>200 Credits</div>
			</div>
			<div class="p-2.5 rounded-lg bg-blue-50/50 border border-blue-100/50">
				<div class="text-lg font-extrabold text-blue-700">12</div>
				<div class="text-[9px] font-bold text-slate-500 uppercase tracking-wide mt-0.5">
					On Track
				</div>
				<div class="text-[8px] font-semibold text-slate-400 mt-0.5">100-200 Credits</div>
			</div>
			<div class="p-2.5 rounded-lg bg-rose-50/50 border border-rose-100/50">
				<div class="text-lg font-extrabold text-rose-700">4</div>
				<div class="text-[9px] font-bold text-slate-500 uppercase tracking-wide mt-0.5">
					Critical
				</div>
				<div class="text-[8px] font-semibold text-slate-400 mt-0.5">&lt;100 Credits</div>
			</div>
		</div>
	</div>
</section>

<!-- ==================== PENDING CERTIFICATE VERIFICATION ==================== -->
<section class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
	<div class="p-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/20">
		<div>
			<h2 class="text-sm font-bold font-serif text-inst-navy">Pending Certificate Verification</h2>
			<p class="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">
				{pendingCertificates.length} certificates awaiting review
			</p>
		</div>
		{#if pendingCertificates.length > 0}
			<span
				class="bg-accent-red text-white text-[10px] font-extrabold uppercase px-2.5 py-1 rounded"
			>
				{pendingCertificates.length} Pending
			</span>
		{/if}
	</div>

	{#if pendingCertificates.length === 0}
		<div class="p-12 text-center flex flex-col items-center justify-center gap-4 bg-slate-50/20">
			<div
				class="w-12 h-12 rounded-full bg-emerald-50 text-emerald-600 flex items-center justify-center border border-emerald-100"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="w-6 h-6"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
				</svg>
			</div>
			<div class="space-y-1">
				<h4 class="text-xs font-bold text-slate-800">All Certificates Processed</h4>
				<p class="text-[11px] text-slate-400 max-w-sm">
					Excellent job! There are no student certificates waiting for review.
				</p>
			</div>
		</div>
	{:else}
		<div class="overflow-x-auto" transition:slide={{ duration: 200 }}>
			<table class="w-full text-left border-collapse">
				<thead>
					<tr
						class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold text-slate-405 uppercase tracking-wider"
					>
						<th class="py-3 px-5">Cert. ID</th>
						<th class="py-3 px-5">Student Name</th>
						<th class="py-3 px-5">Reg. No.</th>
						<th class="py-3 px-5">Certificate Type</th>
						<th class="py-3 px-5">Submitted On</th>
						<th class="py-3 px-5">Priority</th>
						<th class="py-3 px-5 text-center">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100 text-xs font-sans">
					{#each pendingCertificates as cert (cert.id)}
						<tr class="hover:bg-slate-50/50 transition-colors" transition:slide={{ duration: 150 }}>
							<td class="py-3.5 px-5 font-bold text-slate-800 select-all">{cert.id}</td>
							<td class="py-3.5 px-5 font-semibold text-slate-700">{cert.student}</td>
							<td class="py-3.5 px-5 text-slate-500 font-medium uppercase">{cert.regNo}</td>
							<td class="py-3.5 px-5 font-bold text-slate-800">{cert.type}</td>
							<td class="py-3.5 px-5 text-slate-500 font-medium">{cert.date}</td>
							<td class="py-3.5 px-5">
								<span class="inline-flex items-center gap-1.5 font-bold">
									<span
										class="w-2 h-2 rounded-full {cert.priority === 'High'
											? 'bg-rose-600'
											: 'bg-slate-400'}"
									></span>
									<span class={cert.priority === 'High' ? 'text-rose-700' : 'text-slate-500'}>
										{cert.priority}
									</span>
								</span>
							</td>
							<td class="py-3.5 px-5">
								<div class="flex items-center justify-center gap-2">
									<!-- View Button (Matching reference icon & pill style) -->
									<button
										onclick={() => openModal(cert)}
										aria-label="View Certificate"
										class="p-2.5 bg-[#881B1B]/10 hover:bg-[#881B1B]/20 text-[#881B1B] rounded-xl transition-colors focus:outline-none"
									>
										<svg
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
											stroke-width="2"
											stroke="currentColor"
											class="w-4 h-4"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z"
											/>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
											/>
										</svg>
									</button>

									<!-- Approve Button -->
									<button
										onclick={() => handleApprove(cert.id)}
										class="inline-flex items-center justify-center px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white font-extrabold uppercase rounded-lg tracking-wide text-[10px] transition duration-200 focus:outline-none"
									>
										Approve
									</button>

									<!-- Reject Button -->
									<button
										onclick={() => handleReject(cert.id)}
										class="inline-flex items-center justify-center px-3 py-1.5 bg-rose-600 hover:bg-rose-700 text-white font-extrabold uppercase rounded-lg tracking-wide text-[10px] transition duration-200 focus:outline-none"
									>
										Reject
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div
			class="px-5 py-3 border-t border-slate-100 text-[11px] font-bold text-slate-400 uppercase tracking-wider bg-slate-50/20"
		>
			Showing {pendingCertificates.length} of 7 pending certificates
		</div>
	{/if}
</section>

<!-- ==================== DETAILED DIALOG MODAL ==================== -->
{#if isModalOpen && activeCertificate}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			onclick={(e) => e.stopPropagation()}
			class="w-full max-w-xl bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden flex flex-col font-sans"
		>
			<!-- Modal Header -->
			<div class="p-5 border-b border-slate-150 flex items-center justify-between bg-slate-50/30">
				<div>
					<h3 class="text-sm font-bold font-serif text-inst-navy">Review Certificate Submission</h3>
					<p class="text-[9px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">
						ID: {activeCertificate.id}
					</p>
				</div>
				<button
					onclick={closeModal}
					aria-label="Close modal"
					class="p-1 rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Modal Body -->
			<div class="p-6 space-y-5 overflow-y-auto max-h-[60vh]">
				<!-- Student Info Grid -->
				<div class="grid grid-cols-2 gap-4 bg-slate-50 p-4 rounded-xl border border-slate-150">
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Student</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{activeCertificate.student}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Registration No.</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5 uppercase"
							>{activeCertificate.regNo}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Certificate Title</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{activeCertificate.type}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Suggested Credits</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>+{activeCertificate.credits} Credits</span
						>
					</div>
				</div>

				<!-- PDF attachment preview placeholder -->
				<div class="space-y-1.5">
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Attached Document</span
					>
					<div
						class="flex items-center gap-3 p-4 rounded-xl border border-dashed border-slate-300 bg-slate-50/50 hover:bg-slate-50 transition-all select-none"
					>
						<div class="p-2.5 rounded-lg bg-rose-50 text-rose-600 border border-rose-100">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="w-6 h-6"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
								/>
							</svg>
						</div>
						<div class="flex-grow min-w-0">
							<div class="text-xs font-bold text-slate-800 truncate">{activeCertificate.file}</div>
							<div class="text-[10px] font-semibold text-slate-400 mt-0.5">
								PDF Document &middot; 1.4 MB
							</div>
						</div>
						<button
							onclick={() => downloadCertificate(activeCertificate!)}
							class="text-xs font-bold text-accent-red hover:underline focus:outline-none cursor-pointer"
						>
							Download
						</button>
					</div>
				</div>
			</div>

			<!-- Modal Footer -->
			<div class="p-5 border-t border-slate-150 flex items-center justify-end gap-3 bg-slate-50/30">
				<button
					onclick={closeModal}
					class="px-4 py-2 border border-slate-200 hover:bg-slate-50 text-slate-700 font-bold text-xs tracking-wider uppercase rounded-lg transition duration-200 focus:outline-none"
				>
					Close
				</button>
				<button
					onclick={() => handleReject(activeCertificate!.id)}
					class="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white font-bold text-xs tracking-wider uppercase rounded-lg transition duration-200 focus:outline-none"
				>
					Reject Submission
				</button>
				<button
					onclick={() => handleApprove(activeCertificate!.id)}
					class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs tracking-wider uppercase rounded-lg transition duration-200 focus:outline-none"
				>
					Approve Credits
				</button>
			</div>
		</div>
	</div>
{/if}
