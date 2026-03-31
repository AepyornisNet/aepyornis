import { Directive, inject, OnDestroy, signal } from '@angular/core';
import { faMap } from '@ng-icons/font-awesome/regular';
import { faSolidGlobe } from '@ng-icons/font-awesome/solid';
import { _, TranslateService } from '@ngx-translate/core';
import maplibregl, { Map, StyleSpecification } from 'maplibre-gl';

export const OPENFREEMAP_STYLE_BASE_URL = 'https://tiles.openfreemap.org/styles';

export const AERIAL_STYLE: StyleSpecification = {
  version: 8,
  sources: {
    aerial: {
      type: 'raster',
      tiles: [
        'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
      ],
      tileSize: 256,
      attribution: 'Powered by Esri',
    },
  },
  layers: [{ id: 'aerial-layer', type: 'raster', source: 'aerial' }],
};

@Directive()
export abstract class BaseMapComponent implements OnDestroy {
  protected readonly translate = inject(TranslateService);

  protected map?: Map;
  protected navigationControl?: maplibregl.NavigationControl;
  protected baseLayerControl?: maplibregl.IControl;
  protected readonly baseLayerButtons: Partial<Record<'streets' | 'aerial', HTMLButtonElement>> =
    {};
  protected readonly darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  protected styleRefreshVersion = 0;

  public readonly mapStyle = signal<string | StyleSpecification>(this.getStreetStyleUrl());
  public readonly baseLayer = signal<'streets' | 'aerial'>('streets');

  public ngOnDestroy(): void {
    this.darkModeMediaQuery.removeEventListener('change', this.onSystemThemeChanged);
    this.map?.off('style.load', this.onMapStyleLoad);
    this.onDestroy();
  }

  protected onDestroy(): void {
    // Override in child classes for additional cleanup
  }

  protected onMapLoadBase(map: Map): void {
    this.map = map;
    this.darkModeMediaQuery.addEventListener('change', this.onSystemThemeChanged);
    this.map.on('style.load', this.onMapStyleLoad);
    this.addMapControls();
  }

  public setBaseLayer(layer: 'streets' | 'aerial'): void {
    this.baseLayer.set(layer);
    this.updateBaseLayerControlState();
    this.mapStyle.set(layer === 'streets' ? this.getStreetStyleUrl() : AERIAL_STYLE);
    this.refreshAfterStyleChange();
  }

  protected readonly onSystemThemeChanged = (): void => {
    if (this.baseLayer() !== 'streets') {
      return;
    }

    this.mapStyle.set(this.getStreetStyleUrl());
  };

  protected readonly onMapStyleLoad = (): void => {
    this.refreshAfterStyleChange();
  };

  protected abstract refreshAfterStyleChange(): void;

  protected getStreetStyleUrl(): string {
    return `${OPENFREEMAP_STYLE_BASE_URL}/${this.darkModeMediaQuery.matches ? 'fiord' : 'bright'}`;
  }

  protected addMapControls(): void {
    if (!this.map) {
      return;
    }

    if (!this.navigationControl) {
      this.navigationControl = new maplibregl.NavigationControl({
        visualizePitch: true,
        visualizeRoll: true,
        showZoom: true,
        showCompass: true,
      });
      this.map.addControl(this.navigationControl, 'top-right');
    }

    if (!this.baseLayerControl) {
      this.baseLayerControl = this.createBaseLayerControl();
      this.map.addControl(this.baseLayerControl, 'top-right');
    }

    this.updateBaseLayerControlState();
  }

  protected createBaseLayerControl(): maplibregl.IControl {
    const container = document.createElement('div');
    container.className = 'maplibregl-ctrl maplibregl-ctrl-group wt-map-control';

    const streetsButton = this.createControlButton(faMap, _('Streets'), () =>
      this.setBaseLayer('streets'),
    );
    const aerialButton = this.createControlButton(faSolidGlobe, _('Aerial'), () =>
      this.setBaseLayer('aerial'),
    );
    this.baseLayerButtons.streets = streetsButton;
    this.baseLayerButtons.aerial = aerialButton;
    container.append(streetsButton, aerialButton);

    return {
      onAdd: (): HTMLElement => container,
      onRemove: (): void => {
        container.remove();
      },
      getDefaultPosition: () => 'top-right' as const,
    };
  }

  protected createControlButton(
    icon: string,
    title: string,
    onClick: () => void,
  ): HTMLButtonElement {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'wt-map-control-button';
    button.setHTMLUnsafe(icon);
    const translatedTitle = this.translate.instant(title);
    button.title = translatedTitle;
    button.setAttribute('aria-label', translatedTitle);
    button.addEventListener('click', onClick);
    return button;
  }

  protected updateBaseLayerControlState(): void {
    const activeLayer = this.baseLayer();
    for (const [layer, button] of Object.entries(this.baseLayerButtons) as [
      'streets' | 'aerial',
      HTMLButtonElement | undefined,
    ][]) {
      if (!button) {
        continue;
      }
      button.classList.toggle('is-active', layer === activeLayer);
      button.setAttribute('aria-pressed', String(layer === activeLayer));
    }
  }
}
