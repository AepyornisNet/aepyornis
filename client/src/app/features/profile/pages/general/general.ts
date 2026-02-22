import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-general',
  imports: [ReactiveFormsModule, TranslatePipe, AppIcon],
  templateUrl: './general.html',
  styleUrl: './general.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileGeneralPage {
  protected store = inject(ProfileStore);
}
