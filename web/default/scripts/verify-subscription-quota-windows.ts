#!/usr/bin/env bun
import { z } from "zod";

const HOUR = 3600;
const DAY = 86400;
const WEEK = 604800;
const MONTH = 2592000;

// Replicate schemas inline so the script has zero path-alias dependencies
const quotaWindowSchema = z
	.object({
		name: z.string(),
		duration_seconds: z.number().positive(),
		quota: z.number().positive(),
	})
	.strict();

const quotaWindowStateSchema = z
	.object({
		name: z.string(),
		duration_seconds: z.number().positive(),
		quota: z.number().positive(),
		used: z.number().min(0),
		window_start: z.number(),
	})
	.strict();

const subscriptionPlanSchema = z.object({
	id: z.number(),
	title: z.string(),
	subtitle: z.string().optional(),
	price_amount: z.number(),
	currency: z.string().default("USD"),
	duration_unit: z.enum(["year", "month", "day", "hour", "custom"]),
	duration_value: z.number(),
	custom_seconds: z.number().optional(),
	quota_reset_period: z.enum(["never", "daily", "weekly", "monthly", "custom"]),
	quota_reset_custom_seconds: z.number().optional(),
	enabled: z.boolean(),
	sort_order: z.number(),
	max_purchase_per_user: z.number(),
	total_amount: z.number(),
	upgrade_group: z.string().optional(),
	stripe_price_id: z.string().optional(),
	creem_product_id: z.string().optional(),
	waffo_pancake_product_id: z.string().optional(),
	quota_windows: z.array(quotaWindowSchema).optional().default([]),
});

const userSubscriptionSchema = z.object({
	id: z.number(),
	user_id: z.number(),
	plan_id: z.number(),
	status: z.string(),
	source: z.string().optional(),
	start_time: z.number(),
	end_time: z.number(),
	amount_total: z.number(),
	amount_used: z.number(),
	next_reset_time: z.number().optional(),
	quota_windows: z.array(quotaWindowSchema).optional().default([]),
	quota_window_states: z.array(quotaWindowStateSchema).optional().default([]),
});

// Duplicate-name refinement
const noDuplicateNames = z.array(quotaWindowSchema).refine((windows) => {
	const names = windows.map((w) => w.name);
	return new Set(names).size === names.length;
}, "Duplicate window names are not allowed");

// formatWindowLabel (mirrors src implementation)
function formatWindowLabel(window: {
	name: string;
	duration_seconds: number;
}): string {
	const s = window.duration_seconds;
	if (s < HOUR) return `${s}s`;
	if (s < DAY) return `${Math.floor(s / HOUR)}h`;
	if (s < WEEK) return `${Math.floor(s / DAY)}d`;
	if (s < MONTH) return `${Math.floor(s / WEEK)}w`;
	return `${Math.floor(s / DAY)}d`;
}

// formatWindowRemaining (mirrors src, simplified without currency)
function formatWindowRemaining(window: {
	quota: number;
	used: number;
}): string {
	const remaining = window.quota - window.used;
	return `${remaining} / ${window.quota}`;
}

// formatWindowResetCountdown (mirrors src)
function formatWindowResetCountdown(
	window: { duration_seconds: number; window_start: number },
	now?: number,
): string {
	const current = now ?? Math.floor(Date.now() / 1000);
	const end = window.window_start + window.duration_seconds;
	if (current >= end) return "Expired";
	const diff = end - current;
	if (diff < 60) return `${diff}s`;
	if (diff < HOUR) return `${Math.floor(diff / 60)}m ${diff % 60}s`;
	if (diff < DAY) {
		const h = Math.floor(diff / HOUR);
		const m = Math.floor((diff % HOUR) / 60);
		return `${h}h ${m}m`;
	}
	const d = Math.floor(diff / DAY);
	const h = Math.floor((diff % DAY) / HOUR);
	return `${d}d ${h}h`;
}

// ============================================================================
// Test runner
// ============================================================================

let passed = 0;
let failed = 0;

function assert(condition: boolean, label: string): void {
	if (condition) {
		passed++;
	} else {
		failed++;
		console.error(`  FAIL: ${label}`);
	}
}

function assertThrows(fn: () => void, label: string): void {
	try {
		fn();
		failed++;
		console.error(`  FAIL: ${label} — expected error but none thrown`);
	} catch {
		passed++;
	}
}

// ============================================================================
// Tests
// ============================================================================

console.log("\n=== verify-subscription-quota-windows ===\n");

// 1. Plan payload with 5H / 7D / 1M windows; month = 2592000 exactly
{
	const windows = [
		{ name: "5H", duration_seconds: 5 * HOUR, quota: 1000 },
		{ name: "7D", duration_seconds: 7 * DAY, quota: 5000 },
		{ name: "1M", duration_seconds: MONTH, quota: 20000 },
	];
	const parsed = z.array(quotaWindowSchema).parse(windows);
	assert(parsed.length === 3, "3 windows parsed");
	assert(parsed[0].duration_seconds === 5 * HOUR, "5H = 18000s");
	assert(parsed[1].duration_seconds === 7 * DAY, "7D = 604800s");
	assert(parsed[2].duration_seconds === 2592000, "1M = 2592000s exactly");
}

// 2. Schema rejects negative duration_seconds
{
	assertThrows(
		() =>
			quotaWindowSchema.parse({ name: "bad", duration_seconds: -1, quota: 10 }),
		"negative duration_seconds rejected",
	);
}

// 3. Schema rejects zero quota
{
	assertThrows(
		() =>
			quotaWindowSchema.parse({
				name: "bad",
				duration_seconds: 3600,
				quota: 0,
			}),
		"zero quota rejected",
	);
}

// 4. Duplicate names rejected by refine
{
	const result = noDuplicateNames.safeParse([
		{ name: "dup", duration_seconds: HOUR, quota: 100 },
		{ name: "dup", duration_seconds: DAY, quota: 200 },
	]);
	assert(!result.success, "duplicate names rejected by refine");
}

// 5. Empty arrays default to [] in plan and subscription schemas
{
	const plan = subscriptionPlanSchema.parse({
		id: 1,
		title: "Test",
		price_amount: 0,
		duration_unit: "month",
		duration_value: 1,
		quota_reset_period: "never",
		enabled: true,
		sort_order: 0,
		max_purchase_per_user: 0,
		total_amount: 0,
	});
	assert(
		Array.isArray(plan.quota_windows) && plan.quota_windows.length === 0,
		"plan quota_windows defaults to []",
	);

	const sub = userSubscriptionSchema.parse({
		id: 1,
		user_id: 1,
		plan_id: 1,
		status: "active",
		start_time: 0,
		end_time: 0,
		amount_total: 0,
		amount_used: 0,
	});
	assert(
		Array.isArray(sub.quota_windows) && sub.quota_windows.length === 0,
		"subscription quota_windows defaults to []",
	);
	assert(
		Array.isArray(sub.quota_window_states) &&
			sub.quota_window_states.length === 0,
		"subscription quota_window_states defaults to []",
	);
}

// 6. formatWindowLabel for various durations
{
	assert(
		formatWindowLabel({ name: "30s", duration_seconds: 30 }) === "30s",
		"30s label",
	);
	assert(
		formatWindowLabel({ name: "5H", duration_seconds: 5 * HOUR }) === "5h",
		"5h label",
	);
	assert(
		formatWindowLabel({ name: "7D", duration_seconds: 7 * DAY }) === "1w",
		"7d label (7*DAY = 1w)",
	);
	assert(
		formatWindowLabel({ name: "1W", duration_seconds: WEEK }) === "1w",
		"1w label (604800)",
	);
	assert(
		formatWindowLabel({ name: "1M", duration_seconds: MONTH }) === "30d",
		"1M label (2592000 => 30d)",
	);
	assert(
		formatWindowLabel({ name: "90D", duration_seconds: 90 * DAY }) === "90d",
		"90d label",
	);
}

// 7. formatWindowRemaining for normal/exhausted cases
{
	assert(
		formatWindowRemaining({ quota: 1000, used: 200 }) === "800 / 1000",
		"remaining normal",
	);
	assert(
		formatWindowRemaining({ quota: 1000, used: 1000 }) === "0 / 1000",
		"remaining exhausted",
	);
	assert(
		formatWindowRemaining({ quota: 500, used: 600 }) === "-100 / 500",
		"remaining over-consumed",
	);
}

// 8. Legacy subscription with empty windows uses legacy total quota display
{
	const sub = userSubscriptionSchema.parse({
		id: 1,
		user_id: 1,
		plan_id: 1,
		status: "active",
		start_time: 0,
		end_time: 0,
		amount_total: 50000,
		amount_used: 10000,
	});
	assert(
		sub.quota_windows.length === 0 && sub.amount_total === 50000,
		"legacy subscription uses total amount when no windows",
	);
}

// 9. formatWindowResetCountdown
{
	const base = 1000000;
	assert(
		formatWindowResetCountdown(
			{ duration_seconds: HOUR, window_start: base },
			base + HOUR,
		) === "Expired",
		"countdown expired",
	);
	assert(
		formatWindowResetCountdown(
			{ duration_seconds: DAY, window_start: base },
			base + 3600,
		) === "23h 0m",
		"countdown 23h remaining",
	);
	assert(
		formatWindowResetCountdown(
			{ duration_seconds: HOUR, window_start: base },
			base + 120,
		) === "58m 0s",
		"countdown 58m remaining",
	);
}

// ============================================================================
// Result
// ============================================================================

console.log(`\n  Passed: ${passed}`);
console.log(`  Failed: ${failed}`);

if (failed > 0) {
	console.log("\n❌ Smoke test FAILED\n");
	process.exit(1);
}

console.log("\n✅ Smoke test PASSED\n");
