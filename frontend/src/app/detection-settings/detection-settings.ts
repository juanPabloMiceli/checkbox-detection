import {
  Component,
  effect,
  HostListener,
  inject,
  input,
  output,
} from "@angular/core";
import {
  SETTING_GROUPS,
  SettingField,
  DetectorKey,
} from "../models/detection-settings.model";
import { DetectionSettingsStore } from "../services/detection-settings.service";

@Component({
  selector: "app-detection-settings",
  standalone: true,
  templateUrl: "./detection-settings.html",
  styleUrl: "./detection-settings.scss",
})
export class DetectionSettingsPanel {
  readonly visible = input(true);
  readonly processingDisabled = input(false);
  readonly processRequested = output<void>();
  private activeHelp: HTMLElement | null = null;

  constructor() {
    effect(() => {
      if (!this.visible()) this.hideHelp();
    });
  }

  showHelp(event: Event, tooltip: HTMLElement) {
    this.hideHelp();
    const trigger = event.currentTarget as HTMLElement;
    tooltip.showPopover();
    this.activeHelp = tooltip;
    const anchor = trigger.getBoundingClientRect();
    const box = tooltip.getBoundingClientRect();
    const left = Math.max(
      8,
      Math.min(anchor.right - box.width, window.innerWidth - box.width - 8),
    );
    const top =
      anchor.bottom + box.height + 8 <= window.innerHeight
        ? anchor.bottom + 8
        : Math.max(8, anchor.top - box.height - 8);
    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
  }

  @HostListener("window:resize")
  @HostListener("document:keydown.escape")
  hideHelp() {
    this.activeHelp?.hidePopover();
    this.activeHelp = null;
  }

  readonly settings = inject(DetectionSettingsStore);
  readonly closed = output<void>();
  readonly groups = SETTING_GROUPS;
  toggleDetector(key: DetectorKey, event: Event) {
    this.hideHelp();
    this.settings.setDetector(key, (event.target as HTMLInputElement).checked);
  }

  display(field: SettingField) {
    const value = this.settings.draft()[field.key];
    return typeof value === "number"
      ? Number((value * (field.unit === "%" ? 100 : 1)).toPrecision(12))
      : null;
  }
  bound(field: SettingField, bound: "min" | "max") {
    const value = this.settings.definition()?.limits[field.key][bound];
    return value === undefined ? null : value * (field.unit === "%" ? 100 : 1);
  }
  processOnEnter(event: Event) {
    event.preventDefault();
    event.stopPropagation();
    const keyboard = event as KeyboardEvent;
    if (
      keyboard.repeat ||
      keyboard.isComposing ||
      this.processingDisabled() ||
      !this.settings.valid()
    )
      return;
    this.hideHelp();
    this.processRequested.emit();
  }

  change(field: SettingField, event: Event) {
    const input = event.target as HTMLInputElement;
    this.settings.update(
      field.key,
      input.value === "" || !Number.isFinite(input.valueAsNumber)
        ? null
        : input.valueAsNumber / (field.unit === "%" ? 100 : 1),
    );
  }
}
