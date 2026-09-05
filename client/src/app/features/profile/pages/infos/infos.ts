import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-infos',
  imports: [FormField, FormRoot, TranslatePipe],
  templateUrl: './infos.html',
  styleUrl: './infos.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileInfosPage {
  protected store = inject(ProfileStore);
}
