<script lang="ts">
  import { beanLinks } from '$lib/actions/beanLinks';
  import { renderMarkdown } from '$lib/markdown';

  interface Props {
    content: string;
    class?: string;
  }

  let { content, class: className }: Props = $props();

  let renderedHtml = $state('');

  $effect(() => {
    if (content) {
      renderMarkdown(content).then((html) => {
        renderedHtml = html;
      });
    } else {
      renderedHtml = '';
    }
  });
</script>

{#if renderedHtml}
  <div class={className} use:beanLinks>
    {@html renderedHtml}
  </div>
{/if}
