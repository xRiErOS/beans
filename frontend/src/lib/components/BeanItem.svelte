<script lang="ts">
  import type { Bean } from '$lib/beans.svelte';
  import { beansStore } from '$lib/beans.svelte';
  import { backlogDrag } from '$lib/backlogDrag.svelte';
  import { matchesFilter } from '$lib/filter';
  import BeanCard from './BeanCard.svelte';
  import BeanItem from './BeanItem.svelte';

  interface Props {
    bean: Bean;
    /** Parent ID of this bean's sibling group (null = top-level) */
    parentId?: string | null;
    index?: number;
    depth?: number;
    selectedId?: string | null;
    onSelect?: (bean: Bean) => void;
    filterText?: string;
    /** Status of the backlog section this bean is in (for cross-section drag) */
    sectionStatus?: string;
  }

  let {
    bean,
    parentId = null,
    index = 0,
    depth = 0,
    selectedId = null,
    onSelect,
    filterText = '',
    sectionStatus
  }: Props = $props();

  const children = $derived(beansStore.children(bean.id));
  const filteredChildren = $derived(
    filterText ? children.filter((child) => matchesFilter(child, filterText)) : children
  );

  function handleClick(e: MouseEvent) {
    e.stopPropagation();
    onSelect?.(bean);
  }
</script>

<div class="bean-item my-1" role="listitem" data-bean-id={bean.id}>
  <!-- Drop indicator before this card -->
  <div
    class={[
      'mx-1 rounded-full transition-colors',
      backlogDrag.showIndicator(parentId, index, bean.id, sectionStatus) ? 'h-0.5 bg-accent' : 'h-0'
    ]}
  ></div>

  <!-- Drag wrapper; the card inside it is the focusable control. -->
  <div
    class={[
      'rounded transition-all',
      backlogDrag.draggedBeanId === bean.id && 'opacity-40',
      backlogDrag.isReparentTarget(bean.id) && 'ring-2 ring-accent ring-offset-1'
    ]}
    draggable="true"
    ondragstart={(e) => backlogDrag.startDrag(e, bean)}
    ondragend={() => backlogDrag.endDrag()}
    ondragover={(e) => backlogDrag.hoverCard(e, parentId, index, bean.id, sectionStatus)}
    onclick={handleClick}
    role="presentation"
  >
    <BeanCard
      {bean}
      variant="list"
      selected={selectedId === bean.id}
      onclick={() => onSelect?.(bean)}
    />
  </div>

  {#if filteredChildren.length > 0}
    <div
      class="ml-6"
      ondragover={(e) => backlogDrag.hoverList(e, bean.id, filteredChildren.length)}
      ondragleave={(e) => backlogDrag.leaveList(e, e.currentTarget, bean.id)}
      ondrop={(e) => backlogDrag.drop(e, bean.id, filteredChildren)}
      role="list"
    >
      {#each filteredChildren as child, i (child.id)}
        <BeanItem
          bean={child}
          parentId={bean.id}
          index={i}
          depth={depth + 1}
          {selectedId}
          {onSelect}
          {filterText}
          {sectionStatus}
        />
      {/each}

      <!-- Drop indicator at end of children -->
      <div
        class={[
          'mx-1 rounded-full transition-colors',
          backlogDrag.showEndIndicator(bean.id, filteredChildren.length)
            ? 'h-0.5 bg-accent'
            : 'h-0'
        ]}
      ></div>
    </div>
  {/if}
</div>
