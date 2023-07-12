package main

import (
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshallPomXml(t *testing.T) {
	const str = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
	<build>
		<plugins>
			<plugin>
				<groupId>org.springframework.boot</groupId>
				<artifactId>spring-boot-maven-plugin</artifactId>
			</plugin>
		</plugins>
	</build>
</project>`
	var project mavenProject
	if err := xml.Unmarshal([]byte(str), &project); err != nil {
		t.Fatal(err)
	}
	plugins := project.Build.Plugins.Plugins
	assert.Len(t, plugins, 1)
	assert.Equal(t, plugins[0].GroupId, "org.springframework.boot")
	assert.Equal(t, plugins[0].ArtifactId, "spring-boot-maven-plugin")
}
