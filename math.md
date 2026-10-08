# Résumé --- Comment calculer les limites

> Fiche simple pour réviser les **limites**. Les formules sont écrites
> en LaTeX et s'affichent correctement dans les éditeurs Markdown
> compatibles (Typora, Obsidian, VS Code avec extension Markdown/Math,
> etc.).

------------------------------------------------------------------------

## 1. La première chose à faire : remplacer (x)

Pour calculer

$$
\lim_{x\to a} f(x)
$$

commence toujours par essayer de remplacer (x) par (a).

### Exemple

$$
\lim_{x\to2}(3x+1)
$$

On remplace (x) par (2) :

$$
3(2)+1=7
$$

Donc :

$$
\boxed{\lim_{x\to2}(3x+1)=7}
$$

------------------------------------------------------------------------

## 2. Cas d'une fraction

Si tu as :

$$
\lim_{x\to a}\frac{P(x)}{Q(x)}
$$

remplace d'abord (x) par (a).

### Cas normal

Si le dénominateur n'est pas nul, on calcule directement.

Exemple :

$$
\lim_{x\to2}\frac{x+3}{x+1}
=
\frac{2+3}{2+1}
=
\boxed{\frac53}
$$

------------------------------------------------------------------------

## 3. Forme indéterminée (`\frac00`{=tex})

Si le remplacement donne :

$$
\frac00
$$

**ce n'est pas la réponse.**

C'est une **forme indéterminée**.

Il faut transformer l'expression, généralement avec :

-   factorisation ;
-   identités remarquables ;
-   simplification ;
-   conjugué (avec les racines).

------------------------------------------------------------------------

# 4. Factorisation

## Facteur commun

Exemple :

$$
3x^2-12x+12
$$

Tous les termes sont divisibles par (3), donc :

$$
3x^2-12x+12
=
3(x^2-4x+4)
$$

------------------------------------------------------------------------

## 5. Identités remarquables importantes

À connaître :

$$
(a+b)^2=a^2+2ab+b^2
$$

$$
(a-b)^2=a^2-2ab+b^2
$$

$$
(a-b)(a+b)=a^2-b^2
$$

### Exemple

$$
x^2-4x+4
$$

On reconnaît :

$$
x^2-2(x)(2)+2^2
$$

Donc :

$$
\boxed{x^2-4x+4=(x-2)^2}
$$

------------------------------------------------------------------------

# 6. Exemple complet

Calculons :

$$
\lim_{x\to2}
\frac{3x^2-12x+12}{2-x}
$$

### Étape 1 --- Remplacement

Numérateur :

$$
3(2)^2-12(2)+12
$$

$$
=12-24+12=0
$$

Dénominateur :

$$
2-2=0
$$

On obtient :

$$
\frac00
$$

Donc il faut simplifier.

### Étape 2 --- Factoriser le numérateur

$$
3x^2-12x+12
$$

$$
=3(x^2-4x+4)
$$

$$
=3(x-2)^2
$$

La limite devient :

$$
\lim_{x\to2}
\frac{3(x-2)^2}{2-x}
$$

### Étape 3 --- Transformer le dénominateur

On sait que :

$$
2-x=-(x-2)
$$

Donc :

$$
\lim_{x\to2}
\frac{3(x-2)^2}{-(x-2)}
$$

### Étape 4 --- Simplifier

Pour (x`\neq2`{=tex}) :

$$
\frac{3(x-2)^2}{-(x-2)}
=
-3(x-2)
$$

Donc :

$$
\lim_{x\to2}-3(x-2)
$$

### Étape 5 --- Remplacer à nouveau

$$
-3(2-2)=0
$$

Donc :

$$
\boxed{0}
$$

------------------------------------------------------------------------

# 7. Quand il y a une racine : le conjugué

Exemple :

$$
\lim_{x\to4}
\frac{\sqrt{x}-2}{x-4}
$$

Le remplacement donne :

$$
\frac00
$$

On utilise le **conjugué** de

$$
\sqrt{x}-2
$$

qui est :

$$
\sqrt{x}+2
$$

On multiplie en haut et en bas :

$$
\frac{\sqrt{x}-2}{x-4}
\times
\frac{\sqrt{x}+2}{\sqrt{x}+2}
$$

Au numérateur :

$$
(\sqrt{x}-2)(\sqrt{x}+2)=x-4
$$

Donc :

$$
\frac{x-4}{(x-4)(\sqrt{x}+2)}
$$

On simplifie :

$$
\frac{1}{\sqrt{x}+2}
$$

Puis on remplace (x=4) :

$$
\frac{1}{2+2}
=
\boxed{\frac14}
$$

------------------------------------------------------------------------

# 8. Limites vers l'infini --- Polynômes

Pour un polynôme, le terme avec la **plus grande puissance** domine.

Exemple :

$$
\lim_{x\to+\infty}(3x^2-5x+1)
$$

Le terme dominant est :

$$
3x^2
$$

Comme :

$$
3x^2\to+\infty
$$

alors :

$$
\boxed{\lim_{x\to+\infty}(3x^2-5x+1)=+\infty}
$$

------------------------------------------------------------------------

# 9. Fraction à l'infini

Pour :

$$
\lim_{x\to\infty}
\frac{ax^n+\cdots}{bx^m+\cdots}
$$

compare les degrés (n) et (m).

### Si (n\<m)

$$
\boxed{\text{limite}=0}
$$

### Si (n=m)

La limite est le rapport des coefficients dominants :

$$
\boxed{\frac ab}
$$

### Si (n\>m)

La fraction se comporte comme :

$$
\frac ab x^{n-m}
$$

Il faut alors étudier son signe et la direction (+`\infty`{=tex}) ou
(-`\infty`{=tex}).

------------------------------------------------------------------------

# 10. Méthode rapide à retenir

Quand tu vois une limite :

1.  **Remplace (x)** par la valeur demandée.
2.  Si tu obtiens un nombre normal → **terminé**.
3.  Si tu obtiens (`\frac00`{=tex}) → **ne t'arrête pas**.
4.  Cherche une **factorisation** ou une **identité remarquable**.
5.  S'il y a une racine → pense au **conjugué**.
6.  **Simplifie** le facteur responsable du (0/0).
7.  **Remplace à nouveau**.
8.  Pour (x`\to`{=tex}`\infty`{=tex}), cherche les **termes de plus haut
    degré**.

------------------------------------------------------------------------

## Mini mémo

$$
\boxed{
\frac00
\quad\Longrightarrow\quad
\text{Factoriser / Transformer}
\quad\Longrightarrow\quad
\text{Simplifier}
\quad\Longrightarrow\quad
\text{Remplacer à nouveau}
}
$$

### Identités à mémoriser

$$
(a-b)^2=a^2-2ab+b^2
$$

$$
(a+b)^2=a^2+2ab+b^2
$$

$$
(a-b)(a+b)=a^2-b^2
$$

Et n'oublie pas :

$$
a-b=-(b-a)
$$

Par exemple :

$$
\boxed{2-x=-(x-2)}
$$