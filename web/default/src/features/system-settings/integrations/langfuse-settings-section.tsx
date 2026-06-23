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
import * as z from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";
import {
	Form,
	FormControl,
	FormDescription,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
	SettingsForm,
	SettingsSwitchContent,
	SettingsSwitchItem,
} from "../components/settings-form-layout";
import { SettingsPageFormActions } from "../components/settings-page-context";
import { SettingsSection } from "../components/settings-section";
import { useResetForm } from "../hooks/use-reset-form";
import { useUpdateOption } from "../hooks/use-update-option";
import { removeTrailingSlash } from "./utils";

const createLangfuseSchema = (t: (key: string) => string) =>
	z.object({
		LangfuseRequestLogEnabled: z.boolean(),
		LangfuseRequestLogBaseURL: z.string().refine((value) => {
			const trimmed = value.trim();
			if (!trimmed) return true;
			return /^https?:\/\//.test(trimmed);
		}, t("Provide a valid URL starting with http:// or https://")),
		LangfuseRequestLogPublicKey: z.string(),
		LangfuseRequestLogSecretKey: z.string(),
	});

type LangfuseFormValues = z.infer<ReturnType<typeof createLangfuseSchema>>;

type LangfuseSettingsSectionProps = {
	defaultValues: LangfuseFormValues;
};

export function LangfuseSettingsSection({
	defaultValues,
}: LangfuseSettingsSectionProps) {
	const { t } = useTranslation();
	const updateOption = useUpdateOption();
	const langfuseSchema = createLangfuseSchema(t);

	const form = useForm<LangfuseFormValues>({
		resolver: zodResolver(langfuseSchema),
		defaultValues,
	});

	useResetForm(form, defaultValues);

	const onSubmit = async (values: LangfuseFormValues) => {
		const sanitizedUrl = removeTrailingSlash(values.LangfuseRequestLogBaseURL);
		const initialUrl = removeTrailingSlash(
			defaultValues.LangfuseRequestLogBaseURL,
		);
		const sanitizedPublicKey = values.LangfuseRequestLogPublicKey.trim();
		const sanitizedSecretKey = values.LangfuseRequestLogSecretKey.trim();

		const updates: Array<{ key: string; value: string | boolean }> = [];

		if (
			values.LangfuseRequestLogEnabled !==
			defaultValues.LangfuseRequestLogEnabled
		) {
			updates.push({
				key: "LangfuseRequestLogEnabled",
				value: values.LangfuseRequestLogEnabled,
			});
		}

		if (sanitizedUrl !== initialUrl) {
			updates.push({ key: "LangfuseRequestLogBaseURL", value: sanitizedUrl });
		}

		if (
			sanitizedPublicKey &&
			sanitizedPublicKey !== defaultValues.LangfuseRequestLogPublicKey.trim()
		) {
			updates.push({
				key: "LangfuseRequestLogPublicKey",
				value: sanitizedPublicKey,
			});
		}

		if (
			sanitizedSecretKey &&
			sanitizedSecretKey !== defaultValues.LangfuseRequestLogSecretKey.trim()
		) {
			updates.push({
				key: "LangfuseRequestLogSecretKey",
				value: sanitizedSecretKey,
			});
		}

		for (const update of updates) {
			await updateOption.mutateAsync(update);
		}
	};

	return (
		<SettingsSection title={t("Langfuse Request Logging")}>
			<Form {...form}>
				<SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete="off">
					<SettingsPageFormActions
						onSave={form.handleSubmit(onSubmit)}
						isSaving={updateOption.isPending}
						saveLabel="Save Langfuse settings"
					/>
					<FormField
						control={form.control}
						name="LangfuseRequestLogEnabled"
						render={({ field }) => (
							<SettingsSwitchItem>
								<SettingsSwitchContent>
									<FormLabel>{t("Enable Langfuse Request Logging")}</FormLabel>
									<FormDescription>
										{t(
											"When enabled, request traces are exported to the configured Langfuse endpoint.",
										)}
									</FormDescription>
								</SettingsSwitchContent>
								<FormControl>
									<Switch
										checked={field.value}
										onCheckedChange={field.onChange}
									/>
								</FormControl>
							</SettingsSwitchItem>
						)}
					/>

					<FormField
						control={form.control}
						name="LangfuseRequestLogBaseURL"
						render={({ field }) => (
							<FormItem>
								<FormLabel>{t("Langfuse Base URL")}</FormLabel>
								<FormControl>
									<Input
										type="url"
										inputMode="url"
										placeholder={t("https://cloud.langfuse.com")}
										autoComplete="off"
										{...field}
										onChange={(event) => field.onChange(event.target.value)}
									/>
								</FormControl>
								<FormDescription>
									{t(
										"Base URL of your Langfuse instance. Trailing slashes are removed automatically.",
									)}
								</FormDescription>
								<FormMessage />
							</FormItem>
						)}
					/>

					<FormField
						control={form.control}
						name="LangfuseRequestLogPublicKey"
						render={({ field }) => (
							<FormItem>
								<FormLabel>{t("Public Key")}</FormLabel>
								<FormControl>
									<Input
										type="password"
										placeholder={t("Enter new public key to update")}
										autoComplete="new-password"
										{...field}
										onChange={(event) => field.onChange(event.target.value)}
									/>
								</FormControl>
								<FormDescription>
									{t(
										"Langfuse public key for authentication. Leave blank to keep the existing key.",
									)}
								</FormDescription>
								<FormMessage />
							</FormItem>
						)}
					/>

					<FormField
						control={form.control}
						name="LangfuseRequestLogSecretKey"
						render={({ field }) => (
							<FormItem>
								<FormLabel>{t("Secret Key")}</FormLabel>
								<FormControl>
									<Input
										type="password"
										placeholder={t("Enter new secret key to update")}
										autoComplete="new-password"
										{...field}
										onChange={(event) => field.onChange(event.target.value)}
									/>
								</FormControl>
								<FormDescription>
									{t(
										"Langfuse secret key for authentication. Leave blank to keep the existing key.",
									)}
								</FormDescription>
								<FormMessage />
							</FormItem>
						)}
					/>
				</SettingsForm>
			</Form>
		</SettingsSection>
	);
}
