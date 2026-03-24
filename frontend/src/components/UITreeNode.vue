<template>
  <div class="ui-tree-node" :class="{ 'selected': isSelected }" :ref="nodeRef">
    <div class="node-content" @click.stop="handleClick" @mouseenter="handleMouseEnter" @mouseleave="handleMouseLeave">
      <div class="node-indent" :style="{ paddingLeft: (level * 16) + 'px' }"></div>
      <div class="node-toggle" v-if="hasChildren" @click.stop="toggleExpanded">
        <i :class="expanded ? 'pi pi-chevron-down' : 'pi pi-chevron-right'" class="toggle-icon"></i>
      </div>
      <div class="node-info">
        <span class="node-class" :title="nodeDisplayName">{{ nodeDisplayName }}</span>
        <span class="node-children-count">({{ node.children ? node.children.length : 0 }})</span>
      </div>
    </div>
    
    <div v-if="hasChildren && expanded" class="node-children">
      <UITreeNode 
        v-for="child in node.children" 
        :key="child.id"
        :node="child"
        :level="level + 1"
        :selected-id="selectedId"
        :expanded-nodes="expandedNodes"
        @select="$emit('select', $event)"
        @toggle-expanded="(nodeId, isExpanded) => $emit('toggle-expanded', nodeId, isExpanded)"
        @hover="$emit('hover', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue';

const props = defineProps({
  node: {
    type: Object,
    required: true
  },
  level: {
    type: Number,
    default: 0
  },
  selectedId: {
    type: String,
    default: ''
  },
  expandedNodes: {
    type: Object,
    default: () => ({})
  }
});

const emit = defineEmits(['select', 'toggle-expanded', 'hover']);

const nodeRef = ref<any>(null);

const hasChildren = computed(() => {
  return props.node.children && props.node.children.length > 0;
});

const isSelected = computed(() => {
  return props.selectedId === props.node.id;
});

const expanded = computed(() => {
  return !!props.expandedNodes[props.node.id];
});

watch(() => props.selectedId, (newSelectedId) => {
  if (newSelectedId === props.node.id) {
    nextTick(() => {
      nodeRef.value?.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
        inline: 'center'
      });
    });
  }
});

const nodeDisplayName = computed(() => {
  const node = props.node;
  let displayName = node.class || 'Node';
  
  if (node.contentDesc && node.contentDesc.trim() !== '') {
    displayName += `{${node.contentDesc}}`;
  } else if (node.text && node.text.trim() !== '') {
    displayName += `:${node.text}`;
  }
  
  return displayName;
});

const toggleExpanded = () => {
  emit('toggle-expanded', props.node.id, !expanded.value);
};

const handleClick = () => {
  emit('select', props.node);
};

const handleMouseEnter = () => {
  emit('hover', props.node);
};

const handleMouseLeave = () => {
  emit('hover', null);
};
</script>

<style scoped>
.ui-tree-node {
  font-size: 0.85rem;
  line-height: 1.4;
  min-width: 0;
  font-family: "OPPO Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}

.node-content {
  display: flex;
  align-items: flex-start;
  padding: 0.25rem 0;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s ease;
  min-width: 0;
}

.node-content:hover {
  background-color: rgba(59, 130, 246, 0.1);
}

.ui-tree-node.selected > .node-content {
  background-color: rgba(59, 130, 246, 0.2) !important;
}

.ui-tree-node.selected > .node-content .node-class {
  color: var(--primary-color, #3b82f6) !important;
}

.node-indent {
  flex-shrink: 0;
}

.node-toggle {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 4px;
}

.toggle-icon {
  font-size: 0.75rem;
  color: #6b7280;
}

.node-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1;
  min-width: 0;
  white-space: nowrap;
}

.node-class {
  font-weight: 500;
  color: #374151;
  font-size: 0.85rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-children-count {
  color: #6b7280;
  font-size: 0.8rem;
}
</style>
