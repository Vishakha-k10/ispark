<script lang="ts">
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';
	import { fade, slide } from 'svelte/transition';

	// ── Types (structural match with AdminStudentManagementView's Student) ──────
	type Status = 'Active' | 'At Risk' | 'Pending Review' | 'Inactive';

	interface Student {
		id: string;
		name: string;
		regNo: string;
		department: string;
		semester: number;
		creditsEarned: number;
		creditsTarget: number;
		certificates: number;
		pendingCertificates: number;
		activityCount: number;
		status: Status;
		email: string;
		batch: string;
	}

	type ActivityStatus = 'Completed' | 'Pending' | 'Rejected';

	interface Activity {
		id: number;
		name: string;
		category: string;
		date: string;
		credits: number;
		status: ActivityStatus;
	}

	interface AdminNote {
		id: number;
		author?: string;
		role?: string;
		text: string;
		created_at?: string;
	}

	interface Certificate {
		id: number;
		name: string;
		issuer: string;
		date: string;
		credits: number;
		status: ActivityStatus;
	}

	// ── Props ───────────────────────────────────────────────────────────────────
	let {
		student,
		rank,
		cohortSize,
		activities = [],
		certificates = [],
		onBack,
		onToast
	}: {
		student: Student;
		rank: number;
		cohortSize: number;
		activities?: Activity[];
		certificates?: Certificate[];
		onBack: () => void;
		onToast?: (message: string, type?: 'success' | 'danger') => void;
	} = $props();

	function toast(message: string, type: 'success' | 'danger' = 'success') {
		onToast?.(message, type);
	}

	// ── Derived profile stats ────────────────────────────────────────────────────
	const creditPct = $derived(
		Math.min(100, Math.round((student.creditsEarned / student.creditsTarget) * 100))
	);
	const creditsRemaining = $derived(Math.max(0, student.creditsTarget - student.creditsEarned));
	const participationScore = $derived(Math.min(98, 50 + student.activityCount * 2));
	const pendingCerts = $derived(student.pendingCertificates);

	// ── Table State ──────────────────────────────────────────────────────────────
	let activeTab = $state<'activity' | 'certificate'>('activity');
	let searchQuery = $state('');
	let filterCategory = $state('All');
	let filterStatus = $state<'All' | ActivityStatus>('All');
	let selectedActivity = $state<Activity | null>(null);
	let selectedCert = $state<Certificate | null>(null);

	function scrollToHistory(tab: 'activity' | 'certificate') {
		activeTab = tab;
		setTimeout(() => {
			const el = document.getElementById('history-section');
			if (el) {
				el.scrollIntoView({ behavior: 'smooth' });
			}
		}, 50);
	}

	function downloadStudentReport() {
		try {
			const rows: string[][] = [
				['ISPARC STUDENT INDIVIDUAL REPORT'],
				['Generated At', new Date().toLocaleString()],
				[],
				['STUDENT PROFILE'],
				['Roll No / Reg No', student.regNo],
				['Student Name', student.name],
				['Department', student.department],
				['Semester', String(student.semester)],
				['Batch', student.batch || 'Unassigned'],
				['Email', student.email],
				['Status', student.status],
				['Credits Earned', String(student.creditsEarned)],
				['Credits Required', String(student.creditsTarget)],
				['Completion %', `${creditPct}%`],
				[],
				['CERTIFICATES LOG'],
				['ID', 'Certificate Name', 'Issuer / Category', 'Credits', 'Status', 'Date']
			];

			if (certificates.length === 0) {
				rows.push(['No certificates logged']);
			} else {
				for (const c of certificates) {
					rows.push([String(c.id), c.name, c.issuer, String(c.credits), c.status, c.date]);
				}
			}

			rows.push([]);
			rows.push(['ACTIVITIES PARTICIPATION LOG']);
			rows.push(['ID', 'Activity Name', 'Category', 'Credits', 'Status', 'Date']);

			if (activities.length === 0) {
				rows.push(['No activities logged']);
			} else {
				for (const a of activities) {
					rows.push([String(a.id), a.name, a.category, String(a.credits), a.status, a.date]);
				}
			}

			const csvString = rows
				.map((row) => row.map((cell) => `"${(cell || '').replace(/"/g, '""')}"`).join(','))
				.join('\n');

			const blob = new Blob([csvString], { type: 'text/csv;charset=utf-8;' });
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = `${student.regNo}_Student_Report.csv`;
			document.body.appendChild(link);
			link.click();
			link.remove();
			URL.revokeObjectURL(url);

			toast(`Downloaded report for ${student.name}`);
		} catch {
			toast('Failed to download student report', 'danger');
		}
	}

	function downloadActivityReport() {
		try {
			const rows: string[][] = [
				['ISPARC STUDENT ACTIVITY PARTICIPATION REPORT'],
				['Generated At', new Date().toLocaleString()],
				[],
				['STUDENT DETAILS'],
				['Roll No', student.regNo],
				['Student Name', student.name],
				['Department', student.department],
				['Semester', String(student.semester)],
				['Batch', student.batch || 'Unassigned'],
				[],
				['ACTIVITIES PARTICIPATION LOG'],
				['ID', 'Activity Name', 'Category', 'Credits', 'Status', 'Date']
			];

			if (activities.length === 0) {
				rows.push(['No activities logged']);
			} else {
				for (const a of activities) {
					rows.push([String(a.id), a.name, a.category, String(a.credits), a.status, a.date]);
				}
			}

			const csvString = rows
				.map((row) => row.map((cell) => `"${(cell || '').replace(/"/g, '""')}"`).join(','))
				.join('\n');

			const blob = new Blob([csvString], { type: 'text/csv;charset=utf-8;' });
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = `${student.regNo}_Activity_Report.csv`;
			document.body.appendChild(link);
			link.click();
			link.remove();
			URL.revokeObjectURL(url);

			toast(`Activity report downloaded for ${student.name}`);
		} catch {
			toast('Failed to download activity report', 'danger');
		}
	}

	const categories = $derived(['All', ...Array.from(new Set(activities.map((a) => a.category)))]);

	const filteredActivities = $derived(
		activities.filter((a) => {
			const matchSearch =
				searchQuery === '' ||
				a.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.category.toLowerCase().includes(searchQuery.toLowerCase());
			const matchCat = filterCategory === 'All' || a.category === filterCategory;
			const matchStatus = filterStatus === 'All' || a.status === filterStatus;
			return matchSearch && matchCat && matchStatus;
		})
	);

	// ── Mentor Observations ──────────────────────────────────────────────────────
	interface Note {
		id: number;
		author: string;
		role: string;
		badge: string;
		text: string;
	}

	let notes = $state<Note[]>([]);

	onMount(async () => {
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/students/${student.regNo}/observations`, {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				const data = await res.json();
				notes = data.observations.map((obs: AdminNote) => ({
					id: obs.id,
					author: obs.author || 'Admin',
					role: obs.role || 'Admin Staff',
					badge: 'Admin',
					text: obs.text
				}));
			}
		} catch (e) {
			console.error('Failed to load observations', e);
		}
	});

	let composerOpen = $state(false);
	let composerText = $state('');
	let editingId = $state<number | null>(null);

	function openAddNote() {
		editingId = null;
		composerText = '';
		composerOpen = true;
	}

	function openEditNote(note: Note) {
		editingId = note.id;
		composerText = note.text;
		composerOpen = true;
	}

	async function saveNote() {
		const text = composerText.trim();
		if (text === '') return;

		if (editingId !== null) {
			try {
				const token = localStorage.getItem('admin_token');
				const res = await fetch(
					`${API_BASE_URL}/api/admin/students/${student.regNo}/observations/${editingId}`,
					{
						method: 'PUT',
						headers: {
							'Content-Type': 'application/json',
							Authorization: `Bearer ${token}`
						},
						body: JSON.stringify({ text })
					}
				);
				if (res.ok) {
					notes = notes.map((n) => (n.id === editingId ? { ...n, text } : n));
					toast('Observation updated.');
				} else {
					toast('Failed to update observation', 'danger');
				}
			} catch {
				toast('Failed to update observation', 'danger');
			}
			composerOpen = false;
			composerText = '';
			return;
		}

		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/students/${student.regNo}/observations`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ text })
			});

			if (!res.ok) throw new Error('Failed to save observation');

			const data = await res.json();
			notes = [
				...notes,
				{
					id: data.observation.id,
					author: data.observation.author || 'Admin',
					role: data.observation.role || 'Admin Staff',
					badge: 'Admin',
					text: data.observation.text
				}
			];
			toast('Observation added.');

			composerOpen = false;
			composerText = '';
		} catch {
			toast('Failed to save observation', 'danger');
		}
	}

	// ── Helpers ──────────────────────────────────────────────────────────────────
	function semesterLabel(semester: number): string {
		if (!semester) return '—';
		const suffix = semester === 1 ? 'st' : semester === 2 ? 'nd' : semester === 3 ? 'rd' : 'th';
		return `${semester}${suffix} Semester`;
	}

	function initials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.join('')
			.slice(0, 2)
			.toUpperCase();
	}

	function statusChip(status: ActivityStatus): string {
		switch (status) {
			case 'Completed':
				return 'bg-emerald-50 text-emerald-700 border border-emerald-100';
			case 'Pending':
				return 'bg-amber-50 text-amber-700 border border-amber-100';
			case 'Rejected':
				return 'bg-rose-50 text-rose-700 border border-rose-100';
		}
	}

	function statusDot(status: ActivityStatus): string {
		switch (status) {
			case 'Completed':
				return 'bg-emerald-500';
			case 'Pending':
				return 'bg-amber-500';
			case 'Rejected':
				return 'bg-rose-500';
		}
	}

	function studentStatusChip(status: Status): string {
		switch (status) {
			case 'Active':
				return 'bg-emerald-50 text-emerald-700 border border-emerald-100';
			case 'At Risk':
				return 'bg-rose-50 text-rose-700 border border-rose-100';
			case 'Pending Review':
				return 'bg-amber-50 text-amber-700 border border-amber-100';
			case 'Inactive':
				return 'bg-slate-100 text-slate-500 border border-slate-200';
		}
	}
</script>

<div class="space-y-6" transition:fade={{ duration: 150 }}>
	<!-- ── Back link ─────────────────────────────────────────────────────────── -->
	<button
		onclick={onBack}
		class="hover:text-inst-navy inline-flex items-center gap-1.5 text-[11px] font-bold tracking-wider text-slate-500 uppercase transition-colors"
	>
		<svg
			xmlns="http://www.w3.org/2000/svg"
			fill="none"
			viewBox="0 0 24 24"
			stroke-width="2.5"
			stroke="currentColor"
			class="h-3.5 w-3.5"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
		</svg>
		Back to Students
	</button>

	<!-- ── Page Heading ──────────────────────────────────────────────────────── -->
	<div class="flex flex-col gap-1">
		<h1 class="font-serif text-2xl font-bold text-slate-900">Student Details</h1>
		<p class="text-xs font-semibold tracking-wide text-slate-400">
			View complete student extracurricular profile and performance.
		</p>
	</div>

	<!-- ── Profile Header Card ───────────────────────────────────────────────── -->
	<section class="rounded-xl border border-slate-200 bg-white p-5 shadow-xs sm:p-6">
		<div class="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
			<!-- Identity -->
			<div class="flex min-w-0 items-start gap-4">
				<div
					class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full border-2 border-white bg-[#881B1B] font-serif text-xl font-bold text-white shadow-md"
				>
					{initials(student.name)}
				</div>
				<div class="min-w-0">
					<div class="flex flex-wrap items-center gap-2.5">
						<h2 class="font-serif text-lg font-bold text-slate-900">{student.name}</h2>
						<span
							class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[10px] font-extrabold tracking-wide uppercase {studentStatusChip(
								student.status
							)}"
						>
							<span class="h-1.5 w-1.5 rounded-full bg-current opacity-70"></span>
							{student.status}
						</span>
					</div>
					<p class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
						{student.regNo}
					</p>

					<!-- Meta grid -->
					<div class="mt-4 grid grid-cols-2 gap-x-8 gap-y-3 sm:grid-cols-4">
						<div>
							<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
								>Department</span
							>
							<span class="mt-0.5 block text-xs font-bold text-slate-800">{student.department}</span
							>
						</div>
						<div>
							<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
								>Email</span
							>
							<span class="mt-0.5 block text-xs font-bold text-slate-800">{student.email}</span>
						</div>
						<div>
							<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
								>Semester</span
							>
							<span class="mt-0.5 block text-xs font-bold text-slate-800"
								>{semesterLabel(student.semester)}</span
							>
						</div>
						<div>
							<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
								>Batch</span
							>
							<span class="mt-0.5 block text-xs font-bold text-slate-800"
								>{student.batch || '—'}</span
							>
						</div>
					</div>
				</div>
			</div>

			<!-- Actions -->
			<div class="flex shrink-0 flex-wrap items-center gap-2">
				<button
					onclick={() => scrollToHistory('activity')}
					class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-[11px] font-bold text-slate-600 transition-colors hover:bg-slate-50 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3.75 3v11.25A2.25 2.25 0 0 0 6 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0 1 18 16.5h-2.25m-7.5 0h7.5m-7.5 0-1 3m8.5-3 1 3m0 0 .5 1.5m-.5-1.5h-9.5m0 0-.5 1.5M9 11.25v1.5M12 9v3.75m3-6v6"
						/>
					</svg>
					View Activities
				</button>
				<button
					onclick={() => scrollToHistory('certificate')}
					class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-[11px] font-bold text-slate-600 transition-colors hover:bg-slate-50 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M16.5 18.75h-9m9 0a3 3 0 0 1 3 3h-15a3 3 0 0 1 3-3m9 0v-3.375c0-.621-.503-1.125-1.125-1.125h-.871M7.5 18.75v-3.375c0-.621.504-1.125 1.125-1.125h.872m5.007 0H9.497m5.007 0a7.454 7.454 0 0 1-.982-3.172M9.497 14.25a7.454 7.454 0 0 0 .981-3.172M5.25 4.236c-.982.143-1.954.317-2.916.52A6.003 6.003 0 0 0 7.73 9.728M5.25 4.236V4.5c0 2.108.966 3.99 2.48 5.228M5.25 4.236V2.721C7.456 2.41 9.71 2.25 12 2.25c2.291 0 4.545.16 6.75.47v1.516M7.73 9.728a6.726 6.726 0 0 0 2.748 1.35m8.272-6.842V4.5c0 2.108-.966 3.99-2.48 5.228m2.48-5.492a46.32 46.32 0 0 1 2.916.52 6.003 6.003 0 0 1-5.395 4.972m0 0a6.726 6.726 0 0 1-2.749 1.35m0 0a6.772 6.772 0 0 1-3.044 0"
						/>
					</svg>
					View Certificates
				</button>
				<button
					onclick={downloadStudentReport}
					class="inline-flex items-center gap-1.5 rounded-lg bg-[#881B1B] px-3 py-2 text-[11px] font-bold text-white shadow-xs transition-colors hover:bg-[#881B1B]/90 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5"
						/>
					</svg>
					Download Report
				</button>
			</div>
		</div>
	</section>

	<!-- ── Stat Cards ────────────────────────────────────────────────────────── -->
	<section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
		<!-- Total Credits Earned -->
		<div
			class="rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<div class="rounded-lg border border-rose-100 bg-rose-50 p-2 text-rose-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-4 w-4"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-3">
				<span class="font-serif text-2xl font-bold text-slate-900">{student.creditsEarned}</span>
				<h3 class="mt-1 text-xs font-bold tracking-wide text-slate-800">Total Credits Earned</h3>
				<p class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					{creditPct}% Complete
				</p>
			</div>
		</div>

		<!-- Activities Logged -->
		<div
			class="rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<div class="rounded-lg border border-blue-100 bg-blue-50 p-2 text-blue-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-4 w-4"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3.75 3v11.25A2.25 2.25 0 0 0 6 16.5h12M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-16.5 0v11.25m4.5-8.25v6m4.5-9v9m4.5-6v6"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-3">
				<span class="font-serif text-2xl font-bold text-slate-900">{student.activityCount}</span>
				<h3 class="mt-1 text-xs font-bold tracking-wide text-slate-800">Activities Logged</h3>
				<p class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					This academic year
				</p>
			</div>
		</div>

		<!-- Certificates Verified -->
		<div
			class="rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<div class="rounded-lg border border-emerald-100 bg-emerald-50 p-2 text-emerald-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-4 w-4"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-3">
				<span class="font-serif text-2xl font-bold text-slate-900">{student.certificates}</span>
				<h3 class="mt-1 text-xs font-bold tracking-wide text-slate-800">Certificates Verified</h3>
				<p class="mt-0.5 text-[10px] font-bold tracking-wider text-amber-500 uppercase">
					{pendingCerts} pending review
				</p>
			</div>
		</div>

		<!-- Current Rank -->
		<div
			class="rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<div class="rounded-lg border border-amber-100 bg-amber-50 p-2 text-amber-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-4 w-4"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M16.5 18.75h-9m9 0a3 3 0 0 1 3 3h-15a3 3 0 0 1 3-3m9 0v-3.375c0-.621-.503-1.125-1.125-1.125h-.871M7.5 18.75v-3.375c0-.621.504-1.125 1.125-1.125h.872m5.007 0H9.497m5.007 0a7.454 7.454 0 0 1-.982-3.172M9.497 14.25a7.454 7.454 0 0 0 .981-3.172M5.25 4.236c-.982.143-1.954.317-2.916.52A6.003 6.003 0 0 0 7.73 9.728M5.25 4.236V4.5c0 2.108.966 3.99 2.48 5.228M5.25 4.236V2.721C7.456 2.41 9.71 2.25 12 2.25c2.291 0 4.545.16 6.75.47v1.516M7.73 9.728a6.726 6.726 0 0 0 2.748 1.35m8.272-6.842V4.5c0 2.108-.966 3.99-2.48 5.228m2.48-5.492a46.32 46.32 0 0 1 2.916.52 6.003 6.003 0 0 1-5.395 4.972m0 0a6.726 6.726 0 0 1-2.749 1.35m0 0a6.772 6.772 0 0 1-3.044 0"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-3">
				<span class="font-serif text-2xl font-bold text-slate-900">#{rank}</span>
				<h3 class="mt-1 text-xs font-bold tracking-wide text-slate-800">Current Rank</h3>
				<p class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					Among {cohortSize} students
				</p>
			</div>
		</div>
	</section>

	<!-- ── Credit Progress ───────────────────────────────────────────────────── -->
	<section class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs">
		<div class="border-b border-slate-100 bg-slate-50/20 p-5">
			<h2 class="text-inst-navy font-serif text-sm font-bold">Credit Progress</h2>
		</div>
		<div class="space-y-4 p-5">
			<div class="grid grid-cols-3 gap-4">
				<div class="rounded-xl border border-emerald-100 bg-emerald-50/60 py-4 text-center">
					<div class="font-serif text-2xl font-bold text-emerald-600">{student.creditsEarned}</div>
					<div class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
						Credits Earned
					</div>
				</div>
				<div class="border-slate-150 rounded-xl border bg-slate-50 py-4 text-center">
					<div class="font-serif text-2xl font-bold text-slate-800">{student.creditsTarget}</div>
					<div class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
						Credits Required
					</div>
				</div>
				<div class="rounded-xl border border-amber-100 bg-amber-50/60 py-4 text-center">
					<div class="font-serif text-2xl font-bold text-amber-600">{creditsRemaining}</div>
					<div class="mt-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
						Remaining
					</div>
				</div>
			</div>

			<div>
				<div class="mb-1.5 flex items-center justify-between">
					<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
						>{student.creditsEarned} / {student.creditsTarget} Credits</span
					>
					<span class="text-[10px] font-extrabold text-slate-600">{creditPct}% Complete</span>
				</div>
				<div class="h-2.5 w-full overflow-hidden rounded-full bg-slate-100">
					<div
						class="h-full rounded-full bg-gradient-to-r from-[#881B1B] to-rose-500 transition-all duration-500"
						style="width: {creditPct}%"
					></div>
				</div>
				<p class="mt-2 text-[10px] font-semibold text-slate-400">
					{student.name.split(' ')[0]} has completed {creditPct}% of the required {student.creditsTarget}
					credits. Maintaining the current activity pace, on track to complete before the end of the semester.
				</p>
			</div>
		</div>
	</section>

	<!-- ── History (tabs) ────────────────────────────────────────────────────── -->
	<section
		id="history-section"
		class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs"
	>
		<!-- Tabs -->
		<div class="flex items-center gap-6 border-b border-slate-100 px-5 pt-4">
			<button
				onclick={() => (activeTab = 'activity')}
				class="border-b-2 pb-3 text-xs font-bold transition-colors {activeTab === 'activity'
					? 'text-inst-navy border-[#881B1B]'
					: 'border-transparent text-slate-400 hover:text-slate-600'}"
			>
				Activity Participation History
			</button>
			<button
				onclick={() => (activeTab = 'certificate')}
				class="border-b-2 pb-3 text-xs font-bold transition-colors {activeTab === 'certificate'
					? 'text-inst-navy border-[#881B1B]'
					: 'border-transparent text-slate-400 hover:text-slate-600'}"
			>
				Certificate History
			</button>
		</div>

		{#if activeTab === 'activity'}
			<!-- Filter bar -->
			<div class="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3.5">
				<div class="relative flex items-center">
					<span class="pointer-events-none absolute left-3 text-slate-400">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-3.5 w-3.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
							/>
						</svg>
					</span>
					<input
						type="text"
						placeholder="Search activity..."
						bind:value={searchQuery}
						class="focus:border-slate-350 w-48 rounded-lg border border-slate-200 bg-slate-50 py-2 pr-4 pl-8 text-xs text-slate-800 transition-all focus:w-56 focus:bg-white focus:outline-none"
					/>
				</div>

				<select
					bind:value={filterCategory}
					class="focus:border-slate-350 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-[11px] font-bold text-slate-600 focus:outline-none"
				>
					{#each categories as cat}
						<option value={cat}>{cat === 'All' ? 'Category' : cat}</option>
					{/each}
				</select>

				<select
					bind:value={filterStatus}
					class="focus:border-slate-350 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-[11px] font-bold text-slate-600 focus:outline-none"
				>
					<option value="All">Status</option>
					<option value="Completed">Completed</option>
					<option value="Pending">Pending</option>
					<option value="Rejected">Rejected</option>
				</select>
			</div>

			<!-- Activity table -->
			<div class="overflow-x-auto">
				<table class="w-full border-collapse text-left">
					<thead>
						<tr
							class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
						>
							<th class="px-5 py-3">Activity Name</th>
							<th class="px-5 py-3">Category</th>
							<th class="px-5 py-3">Date</th>
							<th class="px-5 py-3">Credits</th>
							<th class="px-5 py-3">Status</th>
							<th class="px-5 py-3 text-center">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 font-sans text-xs">
						{#if filteredActivities.length === 0}
							<tr>
								<td colspan="6" class="py-14 text-center text-xs font-semibold text-slate-400">
									No activities match your filters.
								</td>
							</tr>
						{:else}
							{#each filteredActivities as activity (activity.id)}
								<tr class="transition-colors hover:bg-slate-50/50">
									<td class="px-5 py-3.5 font-bold text-slate-800">{activity.name}</td>
									<td class="px-5 py-3.5">
										<span
											class="rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-[10px] font-bold text-slate-500"
											>{activity.category}</span
										>
									</td>
									<td class="px-5 py-3.5 font-semibold text-slate-500">{activity.date}</td>
									<td class="px-5 py-3.5 font-extrabold text-slate-800">{activity.credits}</td>
									<td class="px-5 py-3.5">
										<span
											class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[10px] font-extrabold tracking-wide uppercase {statusChip(
												activity.status
											)}"
										>
											<span class="h-1.5 w-1.5 rounded-full {statusDot(activity.status)}"></span>
											{activity.status}
										</span>
									</td>
									<td class="px-5 py-3.5 text-center">
										<button
											onclick={() => (selectedActivity = activity)}
											class="text-inst-navy text-[11px] font-bold hover:underline cursor-pointer"
										>
											View Activity
										</button>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		{:else}
			<!-- Certificate table -->
			<div class="overflow-x-auto">
				<table class="w-full border-collapse text-left">
					<thead>
						<tr
							class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
						>
							<th class="px-5 py-3">Certificate</th>
							<th class="px-5 py-3">Issuer</th>
							<th class="px-5 py-3">Date</th>
							<th class="px-5 py-3">Credits</th>
							<th class="px-5 py-3">Status</th>
							<th class="px-5 py-3 text-center">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 font-sans text-xs">
						{#each certificates as cert (cert.id)}
							<tr class="transition-colors hover:bg-slate-50/50">
								<td class="px-5 py-3.5 font-bold text-slate-800">{cert.name}</td>
								<td class="px-5 py-3.5 font-semibold text-slate-600">{cert.issuer}</td>
								<td class="px-5 py-3.5 font-semibold text-slate-500">{cert.date}</td>
								<td class="px-5 py-3.5 font-extrabold text-slate-800">{cert.credits}</td>
								<td class="px-5 py-3.5">
									<span
										class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[10px] font-extrabold tracking-wide uppercase {statusChip(
											cert.status
										)}"
									>
										<span class="h-1.5 w-1.5 rounded-full {statusDot(cert.status)}"></span>
										{cert.status}
									</span>
								</td>
								<td class="px-5 py-3.5 text-center">
									<button
										onclick={() => (selectedCert = cert)}
										class="text-inst-navy text-[11px] font-bold hover:underline cursor-pointer"
									>
										View
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<!-- ── Performance Summary + Quick Insights ──────────────────────────────── -->
	<section class="grid grid-cols-1 items-stretch gap-5 lg:grid-cols-2">
		<!-- Performance Summary -->
		<div
			class="flex flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs"
		>
			<div class="border-b border-slate-100 bg-slate-50/20 p-5">
				<h2 class="text-inst-navy font-serif text-sm font-bold">Performance Summary</h2>
			</div>
			<div class="grid flex-grow grid-cols-1 gap-4 p-5 sm:grid-cols-2">
				<div class="border-slate-150 flex items-center gap-3 rounded-xl border bg-slate-50 p-4">
					<div
						class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-blue-100 text-blue-600"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-4 w-4"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M3.75 3v11.25A2.25 2.25 0 0 0 6 16.5h12M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-16.5 0v11.25m4.5-8.25v6m4.5-9v9m4.5-6v6"
							/>
						</svg>
					</div>
					<div class="min-w-0">
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Most Active Category</span
						>
						<span class="block text-sm font-bold text-slate-900">Technical</span>
					</div>
				</div>

				<div class="border-slate-150 flex items-center gap-3 rounded-xl border bg-slate-50 p-4">
					<div
						class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-yellow-200 bg-yellow-100 text-yellow-600"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-4 w-4"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.562.562 0 0 0-.586 0L6.982 20.54a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.602a.562.562 0 0 1 .321-.988l5.518-.442a.563.563 0 0 0 .475-.345L11.48 3.5Z"
							/>
						</svg>
					</div>
					<div class="min-w-0">
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Highest Credit Activity</span
						>
						<span class="block truncate text-sm font-bold text-slate-900">NPTEL Certification</span>
						<span class="text-[10px] font-semibold text-slate-500">20 Credits</span>
					</div>
				</div>

				<div class="border-slate-150 flex items-center gap-3 rounded-xl border bg-slate-50 p-4">
					<div
						class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-emerald-200 bg-emerald-100 text-emerald-600"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-4 w-4"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941"
							/>
						</svg>
					</div>
					<div class="min-w-0">
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Participation Score</span
						>
						<span class="block text-sm font-bold text-slate-900">{participationScore}%</span>
					</div>
				</div>

				<div class="border-slate-150 flex items-center gap-3 rounded-xl border bg-slate-50 p-4">
					<div
						class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-purple-200 bg-purple-100 text-purple-600"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-4 w-4"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5"
							/>
						</svg>
					</div>
					<div class="min-w-0">
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Last Activity Date</span
						>
						<span class="block text-sm font-bold text-slate-900">24 Jun 2025</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Quick Insights -->
		<div
			class="flex flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs"
		>
			<div class="border-b border-slate-100 bg-slate-50/20 p-5">
				<h2 class="text-inst-navy font-serif text-sm font-bold">Quick Insights</h2>
			</div>
			<div class="flex flex-grow flex-col space-y-2.5 p-4">
				<div
					class="flex items-center gap-2.5 rounded-lg border border-amber-100 bg-amber-50/60 p-3"
				>
					<span class="h-2 w-2 shrink-0 rounded-full bg-amber-500"></span>
					<p class="text-[11px] font-bold text-slate-700">
						{pendingCerts} Certificates Pending Verification
					</p>
				</div>
				<div
					class="flex items-center gap-2.5 rounded-lg border border-emerald-100 bg-emerald-50/60 p-3"
				>
					<span class="h-2 w-2 shrink-0 rounded-full bg-emerald-500"></span>
					<p class="text-[11px] font-bold text-slate-700">Excellent Participation Rate</p>
				</div>
				<div class="flex items-center gap-2.5 rounded-lg border border-blue-100 bg-blue-50/60 p-3">
					<span class="h-2 w-2 shrink-0 rounded-full bg-blue-500"></span>
					<p class="text-[11px] font-bold text-slate-700">Top 10% in Assigned Batch</p>
				</div>
				<div
					class="flex items-center gap-2.5 rounded-lg border border-purple-100 bg-purple-50/60 p-3"
				>
					<span class="h-2 w-2 shrink-0 rounded-full bg-purple-500"></span>
					<p class="text-[11px] font-bold text-slate-700">Consistent Activity Submission</p>
				</div>

				<div class="mt-auto pt-2">
					<div class="mb-1.5 flex items-center justify-between">
						<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
							>Overall Progress</span
						>
						<span class="text-[10px] font-extrabold text-slate-600">{creditPct}%</span>
					</div>
					<div class="h-2 w-full overflow-hidden rounded-full bg-slate-100">
						<div
							class="h-full rounded-full bg-gradient-to-r from-[#881B1B] to-rose-500"
							style="width: {creditPct}%"
						></div>
					</div>
					<button
						onclick={downloadActivityReport}
						class="mt-3 w-full rounded-lg border border-[#881B1B]/20 bg-[#881B1B]/5 py-2 text-[11px] font-bold tracking-wide text-[#881B1B] uppercase transition-colors hover:bg-[#881B1B]/10 cursor-pointer"
					>
						View Full Activity Report →
					</button>
				</div>
			</div>
		</div>
	</section>

	<!-- ── Mentor Observations ───────────────────────────────────────────────── -->
	<section class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs">
		<div
			class="flex items-center justify-between gap-3 border-b border-slate-100 bg-slate-50/20 p-5"
		>
			<h2 class="text-inst-navy flex items-center gap-2 font-serif text-sm font-bold">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="h-4 w-4 text-[#881B1B]"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"
					/>
				</svg>
				Mentor Observations
			</h2>
			<div class="flex shrink-0 items-center gap-2">
				<button
					onclick={() => openEditNote(notes[notes.length - 1])}
					disabled={notes.length === 0}
					class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-[11px] font-bold text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Z"
						/>
					</svg>
					Edit Note
				</button>
				<button
					onclick={openAddNote}
					class="inline-flex items-center gap-1.5 rounded-lg bg-[#881B1B] px-3 py-1.5 text-[11px] font-bold text-white shadow-xs transition-colors hover:bg-[#881B1B]/90"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Add Note
				</button>
			</div>
		</div>

		<div class="space-y-4 p-5">
			{#if composerOpen}
				<div
					transition:slide={{ duration: 150 }}
					class="rounded-xl border border-slate-200 bg-slate-50/40 p-4"
				>
					<textarea
						bind:value={composerText}
						rows="3"
						placeholder="Write an observation about this student..."
						class="focus:border-slate-350 w-full resize-none rounded-lg border border-slate-200 bg-white p-3 text-xs text-slate-700 focus:outline-none"
					></textarea>
					<div class="mt-3 flex items-center justify-end gap-2">
						<button
							onclick={() => {
								composerOpen = false;
								composerText = '';
								editingId = null;
							}}
							class="rounded-lg border border-slate-200 px-3 py-1.5 text-[11px] font-bold text-slate-600 transition-colors hover:bg-slate-100"
						>
							Cancel
						</button>
						<button
							onclick={saveNote}
							class="rounded-lg bg-[#881B1B] px-3 py-1.5 text-[11px] font-bold text-white shadow-xs transition-colors hover:bg-[#881B1B]/90"
						>
							{editingId !== null ? 'Update' : 'Save'} Observation
						</button>
					</div>
				</div>
			{/if}

			{#each notes as note (note.id)}
				<div class="border-slate-150 rounded-xl border p-4">
					<div class="flex items-start gap-3">
						<div
							class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 border-white bg-[#881B1B] font-serif text-xs font-bold text-white shadow-sm"
						>
							{initials(note.author)}
						</div>
						<div class="min-w-0 flex-grow">
							<div class="flex flex-wrap items-center gap-2">
								<span class="text-xs font-bold text-slate-900">{note.author}</span>
								<span
									class="rounded border border-emerald-100 bg-emerald-50 px-1.5 py-0.5 text-[9px] font-extrabold tracking-wide text-emerald-700 uppercase"
									>{note.badge}</span
								>
							</div>
							<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
								>{note.role}</span
							>
							<p class="mt-2 text-xs leading-relaxed font-medium text-slate-600">{note.text}</p>
						</div>
					</div>
				</div>
			{/each}

			<p class="text-[10px] font-bold tracking-wider text-slate-400 uppercase">
				Showing {notes.length} of {notes.length} observations
			</p>
		</div>
	</section>
</div>

<!-- ── Activity Details Modal ───────────────────────────────────────────── -->
{#if selectedActivity}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4"
		transition:fade={{ duration: 150 }}
	>
		<div
			class="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-xl"
			transition:slide={{ duration: 150 }}
		>
			<div class="flex items-center justify-between border-b border-slate-100 pb-4">
				<div>
					<span class="text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
						>Activity Details</span
					>
					<h3 class="text-base font-bold text-slate-900">{selectedActivity.name}</h3>
				</div>
				<button
					aria-label="Close"
					onclick={() => (selectedActivity = null)}
					class="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>
			<div class="mt-4 space-y-3 font-sans text-xs">
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Category</span>
					<span class="font-extrabold text-slate-800">{selectedActivity.category}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Date</span>
					<span class="font-bold text-slate-800">{selectedActivity.date}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Credits Awarded</span>
					<span class="font-extrabold text-emerald-700">{selectedActivity.credits} Credits</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Status</span>
					<span class="font-bold text-slate-800">{selectedActivity.status}</span>
				</div>
			</div>
			<div class="mt-6 flex justify-end">
				<button
					onclick={() => (selectedActivity = null)}
					class="rounded-lg bg-[#881B1B] px-4 py-2 text-xs font-bold text-white hover:bg-[#881B1B]/90 cursor-pointer"
				>
					Close
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ── Certificate Details Modal ────────────────────────────────────────── -->
{#if selectedCert}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4"
		transition:fade={{ duration: 150 }}
	>
		<div
			class="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-xl"
			transition:slide={{ duration: 150 }}
		>
			<div class="flex items-center justify-between border-b border-slate-100 pb-4">
				<div>
					<span class="text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
						>Certificate Verification</span
					>
					<h3 class="text-base font-bold text-slate-900">{selectedCert.name}</h3>
				</div>
				<button
					aria-label="Close"
					onclick={() => (selectedCert = null)}
					class="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>
			<div class="mt-4 space-y-3 font-sans text-xs">
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Issuer / Organization</span>
					<span class="font-extrabold text-slate-800">{selectedCert.issuer}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Issue Date</span>
					<span class="font-bold text-slate-800">{selectedCert.date}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Credits Granted</span>
					<span class="font-extrabold text-emerald-700">{selectedCert.credits} Credits</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Verification Status</span>
					<span class="font-bold text-slate-800">{selectedCert.status}</span>
				</div>
			</div>
			<div class="mt-6 flex justify-end">
				<button
					onclick={() => (selectedCert = null)}
					class="rounded-lg bg-[#881B1B] px-4 py-2 text-xs font-bold text-white hover:bg-[#881B1B]/90 cursor-pointer"
				>
					Close
				</button>
			</div>
		</div>
	</div>
{/if}
