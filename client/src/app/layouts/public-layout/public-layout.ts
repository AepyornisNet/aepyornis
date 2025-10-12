import { Component } from '@angular/core';
import { Header } from '../../core/components/header/header';
import { Footer } from '../../core/components/footer/footer';

@Component({
  selector: 'app-public-layout',
  imports: [Header, Footer],
  templateUrl: './public-layout.html',
  styleUrl: './public-layout.scss'
})
export class PublicLayout {

}
