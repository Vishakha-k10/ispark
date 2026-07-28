<script lang="ts">
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';

	let loading = $state(true);
	let error = $state<string | null>(null);

	let summary = $state({
		assignedBatches: 0,
		totalStudents: 0,
		compliantStudents: 0,
		defaulters: 0
	});

	interface BatchItem {
		name: string;
		students: number;
		pd: number;
		sb: number;
		compliance: number;
		defaulters: number;
		pendingCerts: number;
		status: string;
		notes?: string;
	}

	interface AlertItem {
		tone: string;
		title: string;
		description: string;
		value: string;
		count: number;
		icon?: string;
	}

	interface RequirementItem {
		tone: string;
		label: string;
		count: number;
	}

	interface StudentRosterItem {
		roll_no: string;
		name: string;
		pd_credits: number;
		sb_credits: number;
		compliance_status: string;
	}

	interface RawBatchItem {
		name?: string;
		students?: number;
		total_students?: number;
		totalStudents?: number;
		pd?: number;
		pd_credits?: number;
		pdCredits?: number;
		sb?: number;
		sb_credits?: number;
		sbCredits?: number;
		compliance?: number;
		compliance_percentage?: number;
		defaulters?: number;
		pending_certs?: number;
		pendingCerts?: number;
		status?: string;
		notes?: string;
	}

	let batches = $state<BatchItem[]>([]);
	let alerts = $state<AlertItem[]>([]);
	let requirements = $state<RequirementItem[]>([]);

	const reportIcons: Record<string, string> = {
		batch:
			'M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z',
		student_progress:
			'M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198v-.001c0-.656-.126-1.283-.356-1.858M15 19.128v-.001c0-.656-.126-1.283-.356-1.858A6.002 6.002 0 009 15.75H5.25A3.75 3.75 0 001.5 19.5v.75h13.5v-.75c0-.129-.007-.257-.02-.384M12.75 7.5a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zm6 3a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z',
		compliance:
			'M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z',
		certificate_verification:
			'M4.26 10.147a60.436 60.436 0 00-.491 6.347A48.627 48.627 0 0112 20.904a48.627 48.627 0 018.232-4.41 60.46 60.46 0 00-.491-6.347m-15.482 0a50.57 50.57 0 00-2.658-.813A59.905 59.905 0 0112 3.493a59.902 59.902 0 0110.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.697 50.697 0 0112 13.489a50.702 50.702 0 017.74-3.342M6.75 15a.75.75 0 100-1.5.75.75 0 000 1.5zm0 0v-3.675A55.378 55.378 0 0112 8.443m-7.007 11.55A5.981 5.981 0 006.75 15.75v-1.5'
	};

	let reports = $state([
		{ title: 'Batch Report', type: 'batch' },
		{ title: 'Student Progress Report', type: 'student_progress' },
		{ title: 'Compliance Report', type: 'compliance' },
		{ title: 'Certificate Verification Report', type: 'certificate_verification' }
	]);

	let searchQuery = $state('');
	let filteredBatches = $derived(
		batches.filter((b) => b.name.toLowerCase().includes(searchQuery.trim().toLowerCase()))
	);

	let exportingType = $state<string | null>(null);

	// Fetch Overview
	async function fetchBatchAnalytics() {
		loading = true;
		error = null;
		try {
			const token = localStorage.getItem('admin_token');
			if (!token) throw new Error('Authentication token missing');

			// Added cache: 'no-cache' to bust browser caching
			const res = await fetch(`${API_BASE_URL}/api/admin/batch-analytics`, {
				headers: { Authorization: `Bearer ${token}` },
				cache: 'no-cache'
			});

			if (!res.ok) {
				const errData = await res.json().catch(() => ({}));
				throw new Error(errData.message || errData.error || 'Failed to fetch batch analytics');
			}

			const data = await res.json();
			console.log('API Response received:', data); // <-- Check your browser console!

			// Added fallbacks for both snake_case and camelCase
			summary = {
				assignedBatches: data.summary?.assigned_batches ?? data.summary?.assignedBatches ?? 0,
				totalStudents: data.summary?.total_students ?? data.summary?.totalStudents ?? 0,
				compliantStudents: data.summary?.compliant_students ?? data.summary?.compliantStudents ?? 0,
				defaulters: data.summary?.defaulters ?? 0
			};

			batches = (data.batches || []).map((b: RawBatchItem) => ({
				name: b.name || 'Unknown Batch',
				students: b.students ?? b.total_students ?? b.totalStudents ?? 0,
				pd: b.pd ?? b.pd_credits ?? b.pdCredits ?? 0,
				sb: b.sb ?? b.sb_credits ?? b.sbCredits ?? 0,
				compliance: b.compliance ?? b.compliance_percentage ?? 0,
				defaulters: b.defaulters ?? 0,
				pendingCerts: b.pending_certs ?? b.pendingCerts ?? 0,
				status: b.status || 'Good',
				notes: b.notes || ''
			}));

			if (data.alerts) alerts = data.alerts;
			if (data.requirements) requirements = data.requirements;

			// If you don't want the hardcoded reports to show when API lacks them, uncomment below:
			// reports = data.reports || [];
			if (data.reports) reports = data.reports;
		} catch (err) {
			const e = err as Error;
			console.error('Fetch Error:', e);
			error = e.message || 'Failed to load batch analytics data';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchBatchAnalytics();
	});

	// View Modal
	let viewingBatch = $state<BatchItem | null>(null);
	let loadingDetail = $state(false);
	let studentRoster = $state<StudentRosterItem[]>([]);

	async function openView(batch: BatchItem) {
		viewingBatch = batch;
		loadingDetail = true;
		studentRoster = [];
		try {
			const token = localStorage.getItem('admin_token');
			if (!token) return;
			const res = await fetch(
				`${API_BASE_URL}/api/admin/batch-analytics/${encodeURIComponent(batch.name)}`,
				{
					headers: { Authorization: `Bearer ${token}` }
				}
			);
			if (res.ok) {
				const data = await res.json();
				if (data.students) {
					studentRoster = data.students;
				}
			}
		} catch (e) {
			console.error('Failed to load batch student roster', e);
		} finally {
			loadingDetail = false;
		}
	}

	function closeView() {
		viewingBatch = null;
		studentRoster = [];
	}

	// Edit Modal
	let editingBatch = $state<BatchItem | null>(null);
	let isSaving = $state(false);
	const statusOptions = ['Excellent', 'Good', 'At Risk'];

	function openEdit(batch: BatchItem) {
		editingBatch = { ...batch };
	}

	function closeEdit() {
		editingBatch = null;
	}

	async function saveEdit() {
		if (!editingBatch) return;
		isSaving = true;
		try {
			const token = localStorage.getItem('admin_token');
			if (!token) throw new Error('Authentication token missing');

			const res = await fetch(
				`${API_BASE_URL}/api/admin/batch-analytics/${encodeURIComponent(editingBatch.name)}`,
				{
					method: 'PUT',
					headers: {
						'Content-Type': 'application/json',
						Authorization: `Bearer ${token}`
					},
					body: JSON.stringify({
						status: editingBatch.status,
						notes: editingBatch.notes
					})
				}
			);

			if (!res.ok) {
				const errData = await res.json().catch(() => ({}));
				throw new Error(errData.message || errData.error || 'Failed to update batch');
			}

			// Update local state
			const index = batches.findIndex((b) => b.name === editingBatch!.name);
			if (index !== -1) {
				batches[index] = {
					...batches[index],
					status: editingBatch.status,
					notes: editingBatch.notes
				};
				batches = [...batches];
			}
			closeEdit();
		} catch (err) {
			const e = err as Error;
			alert(e.message || 'Error updating batch');
		} finally {
			isSaving = false;
		}
	}

	// Export Report
	async function exportReport(reportType: string, title: string) {
		exportingType = reportType;
		try {
			const token = localStorage.getItem('admin_token');
			if (!token) throw new Error('Authentication token missing');

			const timestamp = new Date().getTime();

			const res = await fetch(
				`${API_BASE_URL}/api/admin/batch-analytics/export?type=${encodeURIComponent(reportType)}&t=${timestamp}`,
				{
					headers: { Authorization: `Bearer ${token}` },
					cache: 'no-store'
				}
			);

			if (!res.ok) {
				let errMsg = `Export failed (${res.status})`;
				try {
					const errBody = await res.json();
					errMsg = errBody.error || errBody.message || errMsg;
				} catch {
					// intentionally left blank
				}
				throw new Error(errMsg);
			}

			const blob = await res.blob();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;

			const dateStr = new Date().toISOString().split('T')[0];
			a.download = `${title.replace(/\s+/g, '_')}_${dateStr}.csv`;

			document.body.appendChild(a);
			a.click();
			a.remove();
			window.URL.revokeObjectURL(url);
		} catch (err) {
			const e = err as Error;
			alert(e.message || 'Error exporting report');
		} finally {
			exportingType = null;
		}
	}

	// Updated to match the pill badge design in the reference image
	function statusStyles(status: string) {
		if (status === 'Excellent')
			return {
				pill: 'bg-emerald-50/80 text-emerald-800 border border-emerald-200/60',
				dot: 'bg-emerald-600',
				bar: 'bg-emerald-500',
				text: 'text-emerald-800'
			};
		if (status === 'Good')
			return {
				pill: 'bg-blue-50/80 text-blue-700 border border-blue-200/60',
				dot: 'bg-blue-600',
				bar: 'bg-blue-500',
				text: 'text-blue-700'
			};
		return {
			pill: 'bg-amber-50/80 text-amber-900 border border-amber-200/60',
			dot: 'bg-amber-500',
			bar: 'bg-amber-500',
			text: 'text-amber-900'
		};
	}

	function toneClasses(tone: string) {
		const map: Record<string, { iconBg: string; iconText: string; valueText: string }> = {
			rose: { iconBg: 'bg-rose-50', iconText: 'text-[#881B1B]', valueText: 'text-[#881B1B]' },
			amber: { iconBg: 'bg-amber-50', iconText: 'text-amber-600', valueText: 'text-amber-600' },
			blue: { iconBg: 'bg-blue-50', iconText: 'text-blue-600', valueText: 'text-blue-600' },
			emerald: {
				iconBg: 'bg-emerald-50',
				iconText: 'text-emerald-600',
				valueText: 'text-emerald-600'
			}
		};
		return map[tone] ?? map.blue;
	}
</script>

<div class="space-y-6">
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div
				class="flex items-center gap-3 bg-white px-5 py-3 rounded-2xl border border-slate-200 shadow-sm"
			>
				<div
					class="w-5 h-5 border-2 border-[#881B1B] border-t-transparent rounded-full animate-spin"
				></div>
				<span class="text-sm font-semibold text-slate-700">Loading Batch Analytics...</span>
			</div>
		</div>
	{:else if error}
		<div class="bg-rose-50 border border-rose-200 rounded-2xl p-6 text-center">
			<p class="text-sm font-semibold text-rose-800">{error}</p>
			<button
				onclick={fetchBatchAnalytics}
				class="mt-4 px-4 py-2 bg-rose-600 text-white text-xs font-bold rounded-lg hover:bg-rose-700 transition-colors"
			>
				Retry Loading
			</button>
		</div>
	{:else}
		<!-- ================= Summary cards ================= -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
			<!-- Card 1: Assigned Batches -->
			<div
				class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
			>
				<div class="flex items-center justify-between">
					<span class="text-2xl font-bold font-serif text-slate-900">{summary.assignedBatches}</span
					>
					<div class="p-2.5 rounded-xl bg-purple-50 text-purple-600 border border-purple-100">
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
								d="M20.25 14.15v4.25c0 1.094-.787 2.036-1.872 2.18-2.087.277-4.216.42-6.378.42s-4.291-.143-6.378-.42c-1.085-.144-1.872-1.086-1.872-2.18v-4.25m16.5 0a2.18 2.18 0 00.75-1.661V8.706c0-1.081-.768-2.015-1.837-2.175a48.114 48.114 0 00-3.413-.387m4.5 8.006c-.194.165-.42.295-.673.38A23.978 23.978 0 0112 15.75c-2.648 0-5.195-.429-7.577-1.22a2.016 2.016 0 01-.673-.38m0 0A2.18 2.18 0 013 12.489V8.706c0-1.081.768-2.015 1.837-2.175a48.111 48.111 0 013.413-.387m7.5 0V5.25A2.25 2.25 0 0013.5 3h-3a2.25 2.25 0 00-2.25 2.25v.894m7.5 0a48.667 48.667 0 00-7.5 0"
							/>
						</svg>
					</div>
				</div>
				<div class="mt-4">
					<h3 class="text-xs font-bold text-slate-900 capitalize">Assigned Batches</h3>
					<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block"
						>CURRENTLY ACTIVE</span
					>
				</div>
			</div>

			<!-- Card 2: Total Students -->
			<div
				class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
			>
				<div class="flex items-center justify-between">
					<span class="text-2xl font-bold font-serif text-slate-900">{summary.totalStudents}</span>
					<div class="p-2.5 rounded-xl bg-blue-50 text-blue-600 border border-blue-100">
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
					<h3 class="text-xs font-bold text-slate-900 capitalize">Total Students</h3>
					<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block"
						>ACROSS BATCHES</span
					>
				</div>
			</div>

			<!-- Card 3: Compliant Students -->
			<div
				class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
			>
				<div class="flex items-center justify-between">
					<span class="text-2xl font-bold font-serif text-slate-900"
						>{summary.compliantStudents}</span
					>
					<div class="p-2.5 rounded-xl bg-emerald-50 text-emerald-600 border border-emerald-100">
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
								d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					</div>
				</div>
				<div class="mt-4">
					<h3 class="text-xs font-bold text-slate-900 capitalize">Compliant Students</h3>
					<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block"
						>MET REQUIREMENTS</span
					>
				</div>
			</div>

			<!-- Card 4: Defaulters -->
			<div
				class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
			>
				<div class="flex items-center justify-between">
					<span class="text-2xl font-bold font-serif text-[#881B1B]">{summary.defaulters}</span>
					<div class="p-2.5 rounded-xl bg-amber-50 text-amber-600 border border-amber-100">
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
								d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
							/>
						</svg>
					</div>
				</div>
				<div class="mt-4">
					<h3 class="text-xs font-bold text-slate-900 capitalize">Defaulters</h3>
					<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block"
						>NEEDS ATTENTION</span
					>
				</div>
			</div>
		</div>

		<!-- ================= Batch performance table ================= -->
		<div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
			<div
				class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 px-5 py-4 border-b border-slate-200"
			>
				<div>
					<h2 class="font-serif font-bold text-sm text-slate-900">Batch Performance Overview</h2>
					<p class="text-xs text-slate-400 mt-0.5">Performance tracking for assigned batches</p>
				</div>
				<div
					class="flex items-center gap-2 bg-slate-50 border border-slate-200 rounded-lg px-3 py-1.5 w-full sm:w-56"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						class="w-4 h-4 text-slate-400 shrink-0"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
						/>
					</svg>
					<input
						type="text"
						bind:value={searchQuery}
						placeholder="Search batch name..."
						class="bg-transparent text-xs text-slate-800 placeholder-slate-400 focus:outline-none w-full"
					/>
				</div>
			</div>

			<div class="overflow-x-auto">
				<table class="w-full min-w-[900px] text-left">
					<thead>
						<tr class="border-b border-slate-200">
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Batch Name</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Students</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>PD Credits</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>SB Credits</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Compliance %</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Defaulters</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Pending Certs</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Status</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Actions</th
							>
						</tr>
					</thead>
					<tbody>
						{#each filteredBatches as batch}
							{@const styles = statusStyles(batch.status)}
							<tr class="border-b border-slate-100 last:border-b-0 hover:bg-slate-50/60">
								<td class="px-5 py-3.5 text-sm font-bold text-slate-900">{batch.name}</td>
								<td class="px-5 py-3.5 text-sm text-slate-700">{batch.students}</td>
								<td class="px-5 py-3.5 text-sm text-slate-700">{batch.pd}</td>
								<td class="px-5 py-3.5 text-sm text-slate-700">{batch.sb}</td>
								<td class="px-5 py-3.5">
									<div class="flex items-center gap-2">
										<div class="w-20 h-1.5 rounded-full bg-slate-100 overflow-hidden">
											<div
												class="h-full rounded-full {styles.bar}"
												style="width:{batch.compliance}%"
											></div>
										</div>
										<span class="text-xs font-semibold text-slate-800">{batch.compliance}%</span>
									</div>
								</td>
								<td
									class="px-5 py-3.5 text-sm font-bold {batch.defaulters > 0
										? 'text-[#881B1B]'
										: 'text-slate-900'}"
								>
									{batch.defaulters}
								</td>
								<td class="px-5 py-3.5 text-sm text-slate-500">{batch.pendingCerts}</td>
								<td class="px-5 py-3.5">
									<span
										class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-black uppercase tracking-wide {styles.pill}"
									>
										<span class="w-2 h-2 rounded-full shrink-0 {styles.dot}"></span>
										{batch.status}
									</span>
								</td>
								<td class="px-5 py-3.5">
									<div class="flex items-center gap-1.5">
										<button
											type="button"
											onclick={() => openView(batch)}
											aria-label="View {batch.name}"
											class="w-7 h-7 rounded-md bg-[#881B1B]/10 text-[#881B1B] flex items-center justify-center hover:bg-[#881B1B]/20 transition-colors"
										>
											<svg
												xmlns="http://www.w3.org/2000/svg"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2"
												class="w-3.5 h-3.5"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z"
												/>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
												/>
											</svg>
										</button>
										<button
											type="button"
											onclick={() => openEdit(batch)}
											aria-label="Edit {batch.name}"
											class="w-7 h-7 rounded-md bg-[#881B1B]/10 text-[#881B1B] flex items-center justify-center hover:bg-[#881B1B]/20 transition-colors"
										>
											<svg
												xmlns="http://www.w3.org/2000/svg"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2"
												class="w-3.5 h-3.5"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="m16.862 4.487 1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125"
												/>
											</svg>
										</button>
									</div>
								</td>
							</tr>
						{:else}
							<tr>
								<td colspan="9" class="px-5 py-8 text-center text-sm text-slate-400">
									{searchQuery ? `No batches match "${searchQuery}".` : 'No batches found.'}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!-- ================= Mentor alerts ================= -->
		{#if alerts.length > 0}
			<div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-5">
				<h3 class="text-xs font-bold uppercase tracking-wider text-[#881B1B] font-serif">
					Mentor Alerts
				</h3>
				<div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
					{#each alerts as alert}
						{@const t = toneClasses(alert.tone)}
						<div class="flex gap-3 p-3 rounded-xl bg-slate-50 border border-slate-100">
							<div class="w-8 h-8 rounded-md {t.iconBg} flex items-center justify-center shrink-0">
								<svg
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
									class="w-4 h-4 {t.iconText}"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d={alert.icon ||
											'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z'}
									/>
								</svg>
							</div>
							<div>
								<p class="text-xs font-bold text-slate-900">{alert.title}</p>
								<p class="text-xs text-slate-500 mt-0.5">{alert.description}</p>
								<p class="text-xs font-bold mt-1 {t.valueText}">{alert.value}</p>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- ================= Semester requirement status ================= -->
		{#if requirements.length > 0}
			<div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-5">
				<h2 class="font-serif font-bold text-sm text-slate-900">Semester Requirement Status</h2>
				<p class="text-xs text-slate-400 mt-0.5">
					Track student completion across both development tracks
				</p>

				<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 mt-4">
					{#each requirements as req}
						{@const tone =
							req.tone === 'emerald'
								? { bg: 'bg-emerald-50', border: 'border-emerald-200', text: 'text-emerald-600' }
								: req.tone === 'amber'
									? { bg: 'bg-amber-50', border: 'border-amber-200', text: 'text-amber-600' }
									: { bg: 'bg-rose-50', border: 'border-rose-200', text: 'text-[#881B1B]' }}
						<div
							class="rounded-xl border {tone.border} {tone.bg} p-4 hover:shadow-sm transition-shadow"
						>
							<p class="text-xs text-slate-600 leading-snug">{req.label}</p>
							<p class="mt-3 text-2xl font-bold font-serif {tone.text}">{req.count}</p>
							<p class="text-xs text-slate-400">Students</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}
		<!-- ================= Reports ================= -->
		<div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-5">
			<h2 class="font-serif font-bold text-sm text-slate-900">Reports</h2>

			<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 mt-4">
				{#each reports as report}
					{@const iconPath = reportIcons[report.type] || reportIcons.batch}
					<div class="rounded-xl border border-slate-200 bg-slate-50 p-4">
						<div class="flex items-center gap-2.5">
							<div
								class="w-8 h-8 rounded-md bg-[#881B1B]/10 flex items-center justify-center shrink-0"
							>
								<svg
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
									class="w-4 h-4 text-[#881B1B]"
								>
									<path stroke-linecap="round" stroke-linejoin="round" d={iconPath} />
								</svg>
							</div>
							<span class="text-xs font-bold font-serif text-slate-900">{report.title}</span>
						</div>
						<button
							type="button"
							disabled={exportingType === report.type}
							onclick={() => exportReport(report.type, report.title)}
							class="mt-3 w-full flex items-center justify-center gap-2 h-8 rounded-lg bg-slate-900 text-white text-xs font-semibold hover:bg-slate-800 disabled:opacity-50 transition-colors"
						>
							{#if exportingType === report.type}
								<div
									class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"
								></div>
								Exporting...
							{:else}
								<svg
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
									class="w-3.5 h-3.5"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3"
									/>
								</svg>
								Export CSV
							{/if}
						</button>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

<!-- ==================== VIEW MODAL ==================== -->
{#if viewingBatch}
	{@const styles = statusStyles(viewingBatch.status)}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="absolute inset-0 bg-black/40 backdrop-blur-sm" onclick={closeView}></div>

		<div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-2xl overflow-hidden">
			<div class="flex items-start justify-between px-6 py-5 border-b border-slate-200">
				<div>
					<h2 class="text-lg font-bold font-serif text-slate-900">Batch Overview Details</h2>
					<p class="text-[11px] font-bold uppercase tracking-wider text-slate-400 mt-1">
						Batch: {viewingBatch.name}
					</p>
				</div>
				<button
					onclick={closeView}
					aria-label="Close"
					class="text-slate-400 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
				<div class="bg-slate-50 border border-slate-200 rounded-xl p-4 grid grid-cols-2 gap-4">
					<div>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400">Batch Name</p>
						<p class="text-sm font-bold text-slate-900 mt-1">{viewingBatch.name}</p>
					</div>
					<div>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400">Status</p>
						<p class="text-sm font-bold mt-1 {styles.text}">{viewingBatch.status}</p>
					</div>
					<div>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400">PD Credits</p>
						<p class="text-sm font-bold text-slate-900 mt-1">{viewingBatch.pd}</p>
					</div>
					<div>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400">SB Credits</p>
						<p class="text-sm font-bold text-slate-900 mt-1">{viewingBatch.sb}</p>
					</div>
				</div>

				<div
					class="bg-slate-50 border border-slate-200 rounded-xl p-4 grid grid-cols-3 divide-x divide-slate-200 text-center"
				>
					<div>
						<p class="text-2xl font-bold font-serif text-slate-900">{viewingBatch.students}</p>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400 mt-1">
							Students
						</p>
					</div>
					<div>
						<p class="text-2xl font-bold font-serif text-[#881B1B]">{viewingBatch.defaulters}</p>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400 mt-1">
							Defaulters
						</p>
					</div>
					<div>
						<p class="text-2xl font-bold font-serif text-emerald-600">
							{viewingBatch.compliance}%
						</p>
						<p class="text-[10px] font-bold uppercase tracking-wide text-slate-400 mt-1">
							Compliance
						</p>
					</div>
				</div>

				{#if viewingBatch.notes}
					<div class="bg-amber-50/50 border border-amber-200/60 rounded-xl p-4">
						<p class="text-[10px] font-bold uppercase tracking-wide text-amber-800">Mentor Notes</p>
						<p class="text-xs text-slate-700 mt-1 whitespace-pre-wrap">{viewingBatch.notes}</p>
					</div>
				{/if}

				<!-- Student Roster Section -->
				<div>
					<h3 class="text-xs font-bold uppercase tracking-wider text-slate-500 mb-2">
						Student Roster
					</h3>
					{#if loadingDetail}
						<div class="py-6 text-center text-xs text-slate-400">Loading roster...</div>
					{:else if studentRoster.length === 0}
						<div class="py-6 text-center text-xs text-slate-400">
							No student details found for this batch.
						</div>
					{:else}
						<div class="border border-slate-200 rounded-xl overflow-hidden">
							<table class="w-full text-left text-xs">
								<thead class="bg-slate-100 border-b border-slate-200">
									<tr>
										<th class="px-3 py-2 font-bold text-slate-600">Roll No</th>
										<th class="px-3 py-2 font-bold text-slate-600">Name</th>
										<th class="px-3 py-2 font-bold text-slate-600 text-center">PD</th>
										<th class="px-3 py-2 font-bold text-slate-600 text-center">SB</th>
										<th class="px-3 py-2 font-bold text-slate-600">Compliance</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-slate-100">
									{#each studentRoster as student}
										<tr class="hover:bg-slate-50">
											<td class="px-3 py-2 font-semibold text-slate-800">{student.roll_no}</td>
											<td class="px-3 py-2 text-slate-700">{student.name}</td>
											<td class="px-3 py-2 text-center text-slate-700">{student.pd_credits}</td>
											<td class="px-3 py-2 text-center text-slate-700">{student.sb_credits}</td>
											<td class="px-3 py-2">
												<span
													class="px-2 py-0.5 rounded-full text-[10px] font-semibold {student.compliance_status ===
													'Completed Both Tracks'
														? 'bg-emerald-50 text-emerald-700'
														: 'bg-amber-50 text-amber-700'}"
												>
													{student.compliance_status}
												</span>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				</div>
			</div>

			<div class="flex justify-end px-6 py-4 border-t border-slate-200">
				<button
					onclick={closeView}
					class="px-5 py-2 rounded-lg bg-[#881B1B] text-white text-xs font-bold hover:bg-[#881B1B]/90 transition-colors"
				>
					Close Overview
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ==================== EDIT MODAL ==================== -->
{#if editingBatch}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="absolute inset-0 bg-black/40 backdrop-blur-sm" onclick={closeEdit}></div>

		<div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden">
			<div class="flex items-start justify-between px-6 py-5 border-b border-slate-200">
				<div>
					<h2 class="text-lg font-bold font-serif text-slate-900">Manage Batch Details</h2>
					<p class="text-[11px] font-bold uppercase tracking-wider text-slate-400 mt-1">
						Modify status and notes for {editingBatch.name}
					</p>
				</div>
				<button
					onclick={closeEdit}
					aria-label="Close"
					class="text-slate-400 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
				<div>
					<label
						class="text-[10px] font-bold uppercase tracking-wide text-slate-400"
						for="batch-status"
					>
						Batch Status
					</label>
					<select
						id="batch-status"
						bind:value={editingBatch.status}
						class="mt-1.5 w-full px-3 py-2 border border-slate-200 rounded-lg text-sm text-slate-800 focus:outline-none focus:border-slate-400 bg-white"
					>
						{#each statusOptions as option}
							<option value={option}>{option}</option>
						{/each}
					</select>
				</div>

				<div>
					<label
						class="text-[10px] font-bold uppercase tracking-wide text-slate-400"
						for="batch-notes"
					>
						Mentor Notes / Description
					</label>
					<textarea
						id="batch-notes"
						rows="4"
						bind:value={editingBatch.notes}
						placeholder="Add mentor observations or notes for this batch..."
						class="mt-1.5 w-full px-3 py-2 border border-slate-200 rounded-lg text-sm text-slate-800 focus:outline-none focus:border-slate-400 bg-white"
					></textarea>
				</div>
			</div>

			<div class="flex justify-end gap-3 px-6 py-4 border-t border-slate-200">
				<button
					onclick={closeEdit}
					disabled={isSaving}
					class="px-5 py-2 rounded-lg border border-slate-300 text-slate-700 text-xs font-bold hover:bg-slate-50 transition-colors disabled:opacity-50"
				>
					Cancel
				</button>
				<button
					onclick={saveEdit}
					disabled={isSaving}
					class="px-5 py-2 rounded-lg bg-[#881B1B] text-white text-xs font-bold hover:bg-[#881B1B]/90 transition-colors disabled:opacity-50 flex items-center gap-2"
				>
					{#if isSaving}
						<div
							class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"
						></div>
						Saving...
					{:else}
						Save Changes
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}
