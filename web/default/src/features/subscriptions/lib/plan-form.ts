/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { z } from "zod";
import type { TFunction } from "i18next";
import { parseQuotaFromDollars, quotaUnitsToDollars } from "@/lib/format";
import type { SubscriptionPlan, PlanPayload } from "../types";

// ============================================================================
// Window Duration Units
// ============================================================================

const WINDOW_HOUR = 3600;
const WINDOW_DAY = 86400;
const WINDOW_WEEK = 604800;
const WINDOW_MONTH = 2592000;

export const WINDOW_DURATION_UNITS = [
	{ value: "hours", multiplier: WINDOW_HOUR, labelKey: "hours" },
	{ value: "days", multiplier: WINDOW_DAY, labelKey: "days" },
	{ value: "weeks", multiplier: WINDOW_WEEK, labelKey: "weeks" },
	{
		value: "fixed_month",
		multiplier: WINDOW_MONTH,
		labelKey: "fixed 30-day month",
	},
] as const;

export type WindowDurationUnitValue =
	(typeof WINDOW_DURATION_UNITS)[number]["value"];

export function getWindowDurationUnitOptions(t: TFunction) {
	return WINDOW_DURATION_UNITS.map((u) => ({
		value: u.value,
		label: t(u.labelKey),
		multiplier: u.multiplier,
	}));
}

/**
 * Convert `duration_seconds` (from API) back to value + unit for form display.
 * Picks the largest unit that divides evenly.
 */
export function decomposeDurationSeconds(seconds: number): {
	value: number;
	unit: WindowDurationUnitValue;
} {
	for (const u of WINDOW_DURATION_UNITS) {
		if (seconds % u.multiplier === 0) {
			return { value: seconds / u.multiplier, unit: u.value };
		}
	}
	return { value: Math.floor(seconds / WINDOW_DAY), unit: "days" };
}

// ============================================================================
// Form Schema
// ============================================================================

const quotaWindowFormSchema = z.object({
	name: z.string().min(1),
	duration_value: z.coerce.number().min(1),
	duration_unit: z.enum(["hours", "days", "weeks", "fixed_month"]),
	display_quota: z.coerce.number().min(0),
});

export type QuotaWindowFormValues = z.infer<typeof quotaWindowFormSchema>;

export function getPlanFormSchema(t: TFunction) {
	return z.object({
		title: z.string().min(1, t("Please enter plan title")),
		subtitle: z.string().optional(),
		price_amount: z.coerce.number().min(0, t("Please enter amount")),
		duration_unit: z.enum(["year", "month", "day", "hour", "custom"]),
		duration_value: z.coerce.number().min(1),
		custom_seconds: z.coerce.number().min(0).optional(),
		quota_reset_period: z.enum([
			"never",
			"daily",
			"weekly",
			"monthly",
			"custom",
		]),
		quota_reset_custom_seconds: z.coerce.number().min(0).optional(),
		enabled: z.boolean(),
		sort_order: z.coerce.number(),
		max_purchase_per_user: z.coerce.number().min(0),
		total_amount: z.coerce.number().min(0),
		upgrade_group: z.string().optional(),
		stripe_price_id: z.string().optional(),
		creem_product_id: z.string().optional(),
		waffo_pancake_product_id: z.string().optional(),
		quota_windows: z.array(quotaWindowFormSchema).default([]),
	});
}

export type PlanFormValues = z.infer<ReturnType<typeof getPlanFormSchema>>;

export const PLAN_FORM_DEFAULTS: PlanFormValues = {
	title: "",
	subtitle: "",
	price_amount: 0,
	duration_unit: "month",
	duration_value: 1,
	custom_seconds: 0,
	quota_reset_period: "never",
	quota_reset_custom_seconds: 0,
	enabled: true,
	sort_order: 0,
	max_purchase_per_user: 0,
	total_amount: 0,
	upgrade_group: "",
	stripe_price_id: "",
	creem_product_id: "",
	waffo_pancake_product_id: "",
	quota_windows: [],
};

// ============================================================================
// Conversion helpers
// ============================================================================

function getMultiplier(unit: WindowDurationUnitValue): number {
	return (
		WINDOW_DURATION_UNITS.find((u) => u.value === unit)?.multiplier ??
		WINDOW_DAY
	);
}

export function planToFormValues(plan: SubscriptionPlan): PlanFormValues {
	return {
		title: plan.title || "",
		subtitle: plan.subtitle || "",
		price_amount: Number(plan.price_amount || 0),
		duration_unit: plan.duration_unit || "month",
		duration_value: Number(plan.duration_value || 1),
		custom_seconds: Number(plan.custom_seconds || 0),
		quota_reset_period: plan.quota_reset_period || "never",
		quota_reset_custom_seconds: Number(plan.quota_reset_custom_seconds || 0),
		enabled: plan.enabled !== false,
		sort_order: Number(plan.sort_order || 0),
		max_purchase_per_user: Number(plan.max_purchase_per_user || 0),
		total_amount: quotaUnitsToDollars(Number(plan.total_amount || 0)),
		upgrade_group: plan.upgrade_group || "",
		stripe_price_id: plan.stripe_price_id || "",
		creem_product_id: plan.creem_product_id || "",
		waffo_pancake_product_id: plan.waffo_pancake_product_id || "",
		quota_windows: (plan.quota_windows || []).map((w) => {
			const decomposed = decomposeDurationSeconds(w.duration_seconds);
			return {
				name: w.name,
				duration_value: decomposed.value,
				duration_unit: decomposed.unit,
				display_quota: quotaUnitsToDollars(Number(w.quota || 0)),
			};
		}),
	};
}

export function formValuesToPlanPayload(values: PlanFormValues): PlanPayload {
	return {
		plan: {
			...values,
			price_amount: Number(values.price_amount || 0),
			currency: "USD",
			duration_value: Number(values.duration_value || 0),
			custom_seconds: Number(values.custom_seconds || 0),
			quota_reset_period: values.quota_reset_period || "never",
			quota_reset_custom_seconds:
				values.quota_reset_period === "custom"
					? Number(values.quota_reset_custom_seconds || 0)
					: 0,
			sort_order: Number(values.sort_order || 0),
			max_purchase_per_user: Number(values.max_purchase_per_user || 0),
			total_amount: parseQuotaFromDollars(Number(values.total_amount || 0)),
			upgrade_group: values.upgrade_group || "",
			quota_windows: (values.quota_windows || []).map((w) => ({
				name: w.name,
				duration_seconds:
					Number(w.duration_value) * getMultiplier(w.duration_unit),
				quota: parseQuotaFromDollars(Number(w.display_quota || 0)),
			})),
		},
	};
}
